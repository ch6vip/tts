package tts

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"tts/internal/models"
	"tts/internal/tts/microsoft"
)

// SegmentJob 表示一个分段合成任务
type SegmentJob struct {
	ID      string              // 任务 ID
	Index   int                 // 片段索引（用于保持顺序）
	Request models.TTSRequest   // TTS 请求
	Context context.Context     // 请求上下文
}

// SegmentResult 表示分段合成结果
type SegmentResult struct {
	ID        string  // 任务 ID
	Index     int     // 片段索引
	AudioData []byte  // 音频数据
	Error     error   // 错误信息
	Duration  time.Duration // 处理耗时
}

// WorkerPool 音频处理工作池
type WorkerPool struct {
	workers     int                    // worker 数量
	jobs        chan *SegmentJob       // 任务队列
	results     chan *SegmentResult    // 结果队列
	wg          sync.WaitGroup         // 等待 goroutine 完成
	client      *microsoft.Client      // TTS 客户端
	ctx         context.Context        // 上下文
	cancel      context.CancelFunc     // 取消函数
	metrics     *PoolMetrics           // 性能指标
	closed      int32                  // 关闭状态标志(原子操作)
}

// PoolMetrics 工作池性能指标
type PoolMetrics struct {
	TotalJobs      int64
	CompletedJobs  int64
	FailedJobs     int64
	ActiveWorkers  int
	TotalLatency   int64 // 总延迟(纳秒)
	mu             sync.RWMutex
}

// NewWorkerPool 创建新的工作池
func NewWorkerPool(workers int, client *microsoft.Client) *WorkerPool {
	if workers <= 0 {
		workers = 5 // 默认 5 个 worker
	}
	if workers > 50 {
		workers = 50 // 最大限制 50 个 worker
	}
	
	return &WorkerPool{
		workers:  workers,
		jobs:     make(chan *SegmentJob, workers*2), // 缓冲区为 worker 数量的 2 倍
		results:  make(chan *SegmentResult, workers*2), // 增大结果缓冲区
		client:   client,
		metrics:  &PoolMetrics{},
		closed:   0,
	}
}

// Start 启动工作池
func (p *WorkerPool) Start() {
	// 使用后台 context 创建一个可取消的 context,用于控制整个池的生命周期
	p.ctx, p.cancel = context.WithCancel(context.Background())

	logrus.Infof("Starting worker pool with %d workers", p.workers)
	
	// 启动 worker goroutines
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	
	// 更新活跃 worker 数
	p.metrics.mu.Lock()
	p.metrics.ActiveWorkers = p.workers
	p.metrics.mu.Unlock()
}

// worker 工作 goroutine
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()
	
	logrus.Debugf("Worker %d started", id)
	
	for {
		select {
		case <-p.ctx.Done():
			logrus.Debugf("Worker %d stopped", id)
			return
			
		case job, ok := <-p.jobs:
			if !ok {
				logrus.Debugf("Worker %d: job channel closed", id)
				return
			}
			
			// 处理任务
			result := p.processJob(job, id)
			
			// 发送结果
			select {
			case p.results <- result:
			case <-p.ctx.Done():
				return
			}
		}
	}
}

// processJob 处理单个任务
func (p *WorkerPool) processJob(job *SegmentJob, workerID int) *SegmentResult {
	startTime := time.Now()

	result := &SegmentResult{
		ID:    job.ID,
		Index: job.Index,
	}

	logrus.Debugf("Worker %d processing job %s (index: %d)", workerID, job.ID, job.Index)

	// 检查 job context 是否已经取消
	if job.Context.Err() != nil {
		result.Error = fmt.Errorf("job context cancelled before processing segment %d: %w", job.Index, job.Context.Err())
		logrus.Warnf("%v", result.Error)
		p.metrics.mu.Lock()
		p.metrics.FailedJobs++
		p.metrics.mu.Unlock()
		return result
	}

	// 执行 TTS 合成, 使用 job 自己的 context
	resp, err := p.client.SynthesizeSpeech(job.Context, job.Request)
	if err != nil {
		result.Error = fmt.Errorf("worker %d failed to synthesize segment %d: %w", workerID, job.Index, err)
		logrus.Errorf("%v", result.Error)

		// 更新失败指标
		p.metrics.mu.Lock()
		p.metrics.FailedJobs++
		p.metrics.mu.Unlock()

		return result
	}
	
	result.AudioData = resp.AudioContent
	result.Duration = time.Since(startTime)
	
	// 更新完成指标和延迟统计
	p.metrics.mu.Lock()
	p.metrics.CompletedJobs++
	p.metrics.TotalLatency += result.Duration.Nanoseconds()
	p.metrics.mu.Unlock()
	
	logrus.Debugf("Worker %d completed job %s in %v (%d bytes)",
		workerID, job.ID, result.Duration, len(result.AudioData))
	
	return result
}

// Submit 提交任务到工作池
func (p *WorkerPool) Submit(job *SegmentJob) error {
	// 检查是否已关闭
	if atomic.LoadInt32(&p.closed) == 1 {
		return fmt.Errorf("worker pool is closed")
	}

	// 检查 job 的 context 是否已经取消
	if job.Context.Err() != nil {
		return fmt.Errorf("job context cancelled before submission: %w", job.Context.Err())
	}

	// 更新提交指标
	p.metrics.mu.Lock()
	p.metrics.TotalJobs++
	p.metrics.mu.Unlock()

	// 尝试非阻塞提交
	select {
	case p.jobs <- job:
		return nil
	case <-p.ctx.Done(): // 检查池是否已关闭
		return fmt.Errorf("worker pool context cancelled: %w", p.ctx.Err())
	case <-job.Context.Done(): // 检查请求是否已取消
		return fmt.Errorf("job context cancelled during submission: %w", job.Context.Err())
	default:
		// 队列满时记录警告并阻塞等待
		logrus.Warnf("Job queue is full (capacity: %d), blocking submission", cap(p.jobs))
		select {
		case p.jobs <- job:
			return nil
		case <-p.ctx.Done(): // 检查池是否已关闭
			return fmt.Errorf("worker pool context cancelled while waiting: %w", p.ctx.Err())
		case <-job.Context.Done(): // 检查请求是否已取消
			return fmt.Errorf("job context cancelled while waiting for submission: %w", job.Context.Err())
		}
	}
}

// Results 获取结果通道
func (p *WorkerPool) Results() <-chan *SegmentResult {
	return p.results
}

// Close 关闭工作池
func (p *WorkerPool) Close() {
	// 使用原子操作标记已关闭
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		logrus.Warn("Worker pool already closed")
		return
	}
	
	logrus.Info("Closing worker pool...")
	
	// 先取消上下文,通知所有 worker 停止
	if p.cancel != nil {
		p.cancel()
	}
	
	// 关闭任务通道
	close(p.jobs)
	
	// 等待所有 worker 完成(带超时)
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		logrus.Info("All workers stopped gracefully")
	case <-time.After(10 * time.Second):
		logrus.Warn("Timeout waiting for workers to stop")
	}
	
	// 关闭结果通道
	close(p.results)
	
	// 更新活跃 worker 数
	p.metrics.mu.Lock()
	p.metrics.ActiveWorkers = 0
	p.metrics.mu.Unlock()
	
	logrus.Info("Worker pool closed")
}

// GetMetrics 获取性能指标
func (p *WorkerPool) GetMetrics() PoolMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	
	return PoolMetrics{
		TotalJobs:     p.metrics.TotalJobs,
		CompletedJobs: p.metrics.CompletedJobs,
		FailedJobs:    p.metrics.FailedJobs,
		ActiveWorkers: p.metrics.ActiveWorkers,
		TotalLatency:  p.metrics.TotalLatency,
	}
}

// ErrGroupProcessor 使用 errgroup 的简化处理器（适合一次性批量任务）
type ErrGroupProcessor struct {
	client      *microsoft.Client
	concurrency int
}

// NewErrGroupProcessor 创建 errgroup 处理器
func NewErrGroupProcessor(client *microsoft.Client, concurrency int) *ErrGroupProcessor {
	if concurrency <= 0 {
		concurrency = 5
	}
	return &ErrGroupProcessor{
		client:      client,
		concurrency: concurrency,
	}
}

// Process 使用 errgroup 并发处理所有片段
// 注意：需要引入 golang.org/x/sync/errgroup
// 此方法仅用于演示，实际使用需要添加依赖
func (e *ErrGroupProcessor) Process(ctx context.Context, segments []models.TTSRequest) ([][]byte, error) {
	results := make([][]byte, len(segments))
	resultsMu := sync.Mutex{}
	
	// 创建信号量控制并发
	sem := make(chan struct{}, e.concurrency)
	
	var wg sync.WaitGroup
	errChan := make(chan error, len(segments))
	
	for idx, req := range segments {
		wg.Add(1)
		idx := idx
		req := req
		
		go func() {
			defer wg.Done()
			
			// 获取信号量
			sem <- struct{}{}
			defer func() { <-sem }()
			
			// 执行合成
			resp, err := e.client.SynthesizeSpeech(ctx, req)
			if err != nil {
				errChan <- fmt.Errorf("segment %d: %w", idx, err)
				return
			}
			
			// 保存结果
			resultsMu.Lock()
			results[idx] = resp.AudioContent
			resultsMu.Unlock()
		}()
	}
	
	wg.Wait()
	close(errChan)
	
	// 检查是否有错误
	if err := <-errChan; err != nil {
		return nil, err
	}
	
	return results, nil
}

// PoolStats 工作池统计信息
type PoolStats struct {
	TotalJobs      int64         `json:"total_jobs"`
	CompletedJobs  int64         `json:"completed_jobs"`
	FailedJobs     int64         `json:"failed_jobs"`
	ActiveWorkers  int           `json:"active_workers"`
	QueueLength    int           `json:"queue_length"`
	SuccessRate    float64       `json:"success_rate"`
}

// Stats 获取详细统计信息
func (p *WorkerPool) Stats() PoolStats {
	metrics := p.GetMetrics()
	
	successRate := 0.0
	if metrics.TotalJobs > 0 {
		successRate = float64(metrics.CompletedJobs) / float64(metrics.TotalJobs) * 100
	}
	
	return PoolStats{
		TotalJobs:     metrics.TotalJobs,
		CompletedJobs: metrics.CompletedJobs,
		FailedJobs:    metrics.FailedJobs,
		ActiveWorkers: metrics.ActiveWorkers,
		QueueLength:   len(p.jobs),
		SuccessRate:   successRate,
	}
}

// GetAverageLatency 获取平均延迟
func (p *WorkerPool) GetAverageLatency() time.Duration {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	
	if p.metrics.CompletedJobs == 0 {
		return 0
	}
	return time.Duration(p.metrics.TotalLatency / p.metrics.CompletedJobs)
}