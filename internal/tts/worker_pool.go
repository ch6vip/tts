package tts

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
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
	logger      zerolog.Logger         // 日志记录器
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
func NewWorkerPool(workers int, client *microsoft.Client, logger zerolog.Logger) *WorkerPool {
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
		logger:   logger,
	}
}

// Start 启动工作池
func (p *WorkerPool) Start() {
	// 使用后台 context 创建一个可取消的 context,用于控制整个池的生命周期
	p.ctx, p.cancel = context.WithCancel(context.Background())

	p.logger.Info().Int("workers", p.workers).Msg("Starting worker pool")
	
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
	
	p.logger.Debug().Int("worker_id", id).Msg("Worker started")
	
	for {
		select {
		case <-p.ctx.Done():
			p.logger.Debug().Int("worker_id", id).Msg("Worker stopped")
			return
			
		case job, ok := <-p.jobs:
			if !ok {
				p.logger.Debug().Int("worker_id", id).Msg("Worker: job channel closed")
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

	p.logger.Debug().
		Int("worker_id", workerID).
		Str("job_id", job.ID).
		Int("index", job.Index).
		Msg("Worker processing job")

	// 检查 job context 是否已经取消
	if job.Context.Err() != nil {
		result.Error = fmt.Errorf("job context cancelled before processing segment %d: %w", job.Index, job.Context.Err())
		p.logger.Warn().Err(result.Error).Msg("Job context cancelled before processing")
		p.metrics.mu.Lock()
		p.metrics.FailedJobs++
		p.metrics.mu.Unlock()
		return result
	}

	// 执行 TTS 合成, 使用 job 自己的 context
	resp, err := p.client.SynthesizeSpeech(job.Context, job.Request)
	if err != nil {
		result.Error = fmt.Errorf("worker %d failed to synthesize segment %d: %w", workerID, job.Index, err)
		p.logger.Error().Err(result.Error).Msg("Worker failed to synthesize segment")

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
	
	p.logger.Debug().
		Int("worker_id", workerID).
		Str("job_id", job.ID).
		Dur("duration", result.Duration).
		Int("bytes", len(result.AudioData)).
		Msg("Worker completed job")
	
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
		p.logger.Warn().Int("capacity", cap(p.jobs)).Msg("Job queue is full, blocking submission")
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
		p.logger.Warn().Msg("Worker pool already closed")
		return
	}
	
	p.logger.Info().Msg("Closing worker pool...")
	
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
		p.logger.Info().Msg("All workers stopped gracefully")
	case <-time.After(10 * time.Second):
		p.logger.Warn().Msg("Timeout waiting for workers to stop")
	}
	
	// 关闭结果通道
	close(p.results)
	
	// 更新活跃 worker 数
	p.metrics.mu.Lock()
	p.metrics.ActiveWorkers = 0
	p.metrics.mu.Unlock()
	
	p.logger.Info().Msg("Worker pool closed")
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