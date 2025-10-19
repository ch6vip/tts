package tts

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"tts/internal/models"
	"tts/internal/tts/audio"
	"tts/internal/tts/microsoft"
)

// LongTextTTSService 长文本 TTS 服务
type LongTextTTSService struct {
	client       *microsoft.Client
	segmenter    SegmentationStrategy
	merger       audio.Merger
	workerPool   *WorkerPool
	maxSegmentLen int
	minTextForSplit int // 触发分段的最小文本长度
}

// LongTextConfig 长文本处理配置
type LongTextConfig struct {
	MaxSegmentLength int    // 每个片段的最大字符数
	WorkerCount      int    // 并发 worker 数量
	MinTextForSplit  int    // 触发分段的最小文本长度
	FFmpegPath       string // FFmpeg 可执行文件路径
	UseSmartSegment  bool   // 是否使用智能分段
}

// NewLongTextTTSService 创建长文本 TTS 服务
func NewLongTextTTSService(client *microsoft.Client, config LongTextConfig) *LongTextTTSService {
	// 设置默认值
	if config.MaxSegmentLength <= 0 {
		config.MaxSegmentLength = 500
	}
	if config.WorkerCount <= 0 {
		config.WorkerCount = 5
	}
	if config.MinTextForSplit <= 0 {
		config.MinTextForSplit = 1000 // 1000 字符以下不分段
	}

	// 选择分段策略
	var segmenter SegmentationStrategy
	if config.UseSmartSegment {
		segmenter = NewSmartSegmenter()
		logrus.Info("Using smart segmentation strategy")
	} else {
		segmenter = NewFixedLengthSegmenter()
		logrus.Info("Using fixed-length segmentation strategy")
	}

	// 创建音频合并器
	merger := audio.NewFFmpegMerger(config.FFmpegPath)

	// 创建并启动工作池
	pool := NewWorkerPool(config.WorkerCount, client)
	pool.Start()

	return &LongTextTTSService{
		client:          client,
		segmenter:       segmenter,
		merger:          merger,
		workerPool:      pool,
		maxSegmentLen:   config.MaxSegmentLength,
		minTextForSplit: config.MinTextForSplit,
	}
}

// SynthesizeSpeech 合成语音（智能判断是否需要分段）
func (s *LongTextTTSService) SynthesizeSpeech(ctx context.Context, req models.TTSRequest) (*models.TTSResponse, error) {
	startTime := time.Now()

	// 检查 context
	if ctx.Err() != nil {
		return nil, fmt.Errorf("context cancelled before synthesis: %w", ctx.Err())
	}

	// 如果文本长度小于阈值，直接调用单次合成
	textLen := utf8.RuneCountInString(req.Text)
	if textLen <= s.minTextForSplit {
		logrus.Infof("Text length (%d) below split threshold (%d), using single synthesis",
			textLen, s.minTextForSplit)
		return s.client.SynthesizeSpeech(ctx, req)
	}

	// 长文本，使用分段合成
	logrus.Infof("Text length (%d) exceeds threshold, using segmented synthesis", textLen)
	return s.synthesizeLongText(ctx, req, startTime)
}

// synthesizeLongText 长文本分段合成
func (s *LongTextTTSService) synthesizeLongText(ctx context.Context, req models.TTSRequest, startTime time.Time) (*models.TTSResponse, error) {
	// 1. 文本分段
	segmentStart := time.Now()
	segments := s.segmenter.Segment(req.Text, s.maxSegmentLen)
	segmentDuration := time.Since(segmentStart)
	
	logrus.Infof("Text segmented into %d parts in %v", len(segments), segmentDuration)

	// 如果只有一个片段，直接合成
	if len(segments) == 1 {
		return s.client.SynthesizeSpeech(ctx, req)
	}

	// 2. 并发提交任务和收集结果
	return s.processSegmentsConcurrently(ctx, req, segments, startTime, segmentDuration)
}

// processSegmentsConcurrently 并发处理文本片段
func (s *LongTextTTSService) processSegmentsConcurrently(ctx context.Context, req models.TTSRequest, segments []string, startTime time.Time, segmentDuration time.Duration) (*models.TTSResponse, error) {
	segmentCount := len(segments)
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())
	
	// 创建通道用于任务提交和结果收集
	submitDone := make(chan struct{})
	resultChan := make(chan *SegmentResult, segmentCount) // 使用缓冲通道提高效率
	errChan := make(chan error, 1) // 用于传递错误
	
	// 使用 WaitGroup 来跟踪所有 goroutine 的完成
	var wg sync.WaitGroup
	wg.Add(2) // 一个用于提交任务，一个用于收集结果
	
	// 初始化音频片段数组
	audioSegments := make([][]byte, segmentCount)
	var totalAudioSize int64
	var mu sync.Mutex // 保护共享变量
	
	// 启动任务提交 goroutine
	go func() {
		defer wg.Done()
		defer close(submitDone)
		
		for idx, segment := range segments {
			// 检查 context
			if ctx.Err() != nil {
				logrus.Warnf("Context cancelled during job submission at segment %d: %v", idx, ctx.Err())
				errChan <- fmt.Errorf("context cancelled during job submission: %w", ctx.Err())
				return
			}
			
			job := &SegmentJob{
				ID:      fmt.Sprintf("%s_seg_%d", jobID, idx),
				Index:   idx,
				Context: ctx, // 传递请求上下文
				Request: models.TTSRequest{
					Text:  segment,
					Voice: req.Voice,
					Rate:  req.Rate,
					Pitch: req.Pitch,
					Style: req.Style,
					Format: req.Format, // 确保包含格式参数
					SSML:  "", // 分段时使用 Text，不使用 SSML
				},
			}

			if err := s.workerPool.Submit(job); err != nil {
				logrus.Errorf("Failed to submit job %d: %v", idx, err)
				errChan <- fmt.Errorf("failed to submit job %d: %w", idx, err)
				return
			}
		}
	}()
	
	// 启动结果收集 goroutine
	go func() {
		defer wg.Done()
		defer close(resultChan)
		
		receivedCount := 0
		
		// 使用结果收集循环，直到收到所有提交任务的结果或 context 取消
		for receivedCount < segmentCount {
			select {
			case result, ok := <-s.workerPool.Results():
				if !ok {
					// 结果通道已关闭，但没有收到所有结果
					errChan <- fmt.Errorf("result channel closed unexpectedly, received %d/%d results",
						receivedCount, segmentCount)
					return
				}
				
				// 将结果转发到结果通道
				select {
				case resultChan <- result:
				case <-ctx.Done():
					errChan <- fmt.Errorf("context cancelled during result forwarding: %w", ctx.Err())
					return
				}
				
				receivedCount++
				
			case <-ctx.Done():
				errChan <- fmt.Errorf("context cancelled during result collection: %w", ctx.Err())
				return
			}
		}
	}()
	
	// 主 goroutine 处理结果
	collectStart := time.Now()
	errorCount := 0
	var firstError error
	receivedCount := 0
	
	// 等待任务提交完成
	select {
	case <-submitDone:
		// 任务提交完成
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled during job submission: %w", ctx.Err())
	case err := <-errChan:
		return nil, err
	}
	
	submitDuration := time.Since(startTime) - segmentDuration
	logrus.Infof("Submitted %d jobs in %v", segmentCount, submitDuration)
	
	// 处理结果
	for receivedCount < segmentCount {
		select {
		case result, ok := <-resultChan:
			if !ok {
				// 结果通道已关闭，但没有收到所有结果
				return nil, fmt.Errorf("result channel closed unexpectedly, received %d/%d results",
					receivedCount, segmentCount)
			}
			
			if result.Error != nil {
				errorCount++
				if firstError == nil {
					firstError = result.Error
				}
				logrus.Errorf("Segment %d failed: %v", result.Index, result.Error)
			} else {
				if result.Index < 0 || result.Index >= len(audioSegments) {
					return nil, fmt.Errorf("invalid segment index: %d", result.Index)
				}
				
				// 使用互斥锁保护共享变量
				mu.Lock()
				audioSegments[result.Index] = result.AudioData
				totalAudioSize += int64(len(result.AudioData))
				mu.Unlock()
				
				logrus.Debugf("Received segment %d/%d (%d bytes, took %v)",
					result.Index+1, segmentCount, len(result.AudioData), result.Duration)
			}
			receivedCount++
		
		case err := <-errChan:
			return nil, err
			
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled during result processing: %w", ctx.Err())
		}
	}
	
	// 等待所有 goroutine 完成
	wg.Wait()
	
	collectDuration := time.Since(collectStart)
	logrus.Infof("Collected %d audio segments (%d bytes total) in %v",
		segmentCount, totalAudioSize, collectDuration)

	// 检查是否有错误
	if errorCount > 0 {
		return nil, fmt.Errorf("synthesis failed: %d/%d segments failed, first error: %w",
			errorCount, segmentCount, firstError)
	}
	
	// 验证所有片段都已收集
	for idx, segment := range audioSegments {
		if segment == nil {
			return nil, fmt.Errorf("missing audio segment at index %d", idx)
		}
	}

	// 3. 合并音频
	mergeStart := time.Now()
	merged, err := s.merger.Merge(audioSegments)
	if err != nil {
		return nil, fmt.Errorf("failed to merge audio segments: %w", err)
	}

	mergeDuration := time.Since(mergeStart)
	totalDuration := time.Since(startTime)
	
	logrus.Infof("Long text synthesis completed in %v (segment: %v, submit: %v, collect: %v, merge: %v)",
		totalDuration, segmentDuration, submitDuration, collectDuration, mergeDuration)

	// 4. 获取工作池统计
	stats := s.workerPool.Stats()
	avgLatency := s.workerPool.GetAverageLatency()
	logrus.Infof("Worker pool stats: %d total, %d completed, %d failed, %.2f%% success rate, avg latency: %v",
		stats.TotalJobs, stats.CompletedJobs, stats.FailedJobs, stats.SuccessRate, avgLatency)

	return &models.TTSResponse{
		AudioContent: merged,
		ContentType:  "audio/mpeg",
		CacheHit:     false,
	}, nil
}

// GetStats 获取服务统计信息
func (s *LongTextTTSService) GetStats() PoolStats {
	return s.workerPool.Stats()
}

// Close 关闭服务（释放资源）
func (s *LongTextTTSService) Close() {
	if s.workerPool != nil {
		s.workerPool.Close()
	}
}