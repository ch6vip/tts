package tts

import (
	"context"
	"fmt"
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

	// 2. 提交所有任务
	submitStart := time.Now()
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())
	
	for idx, segment := range segments {
		// 检查 context
		if ctx.Err() != nil {
			return nil, fmt.Errorf("context cancelled during job submission: %w", ctx.Err())
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
				SSML:  "", // 分段时使用 Text，不使用 SSML
			},
		}

		if err := s.workerPool.Submit(job); err != nil {
			return nil, fmt.Errorf("failed to submit job %d: %w", idx, err)
		}
	}
	submitDuration := time.Since(submitStart)
	logrus.Infof("Submitted %d jobs in %v", len(segments), submitDuration)

	// 4. 收集结果(使用 map 避免重复分配数组)
	collectStart := time.Now()
	audioSegments := make([][]byte, len(segments))
	var totalAudioSize int64
	errorCount := 0
	var firstError error

	for i := 0; i < len(segments); i++ {
		select {
		case result := <-s.workerPool.Results():
			if result.Error != nil {
				errorCount++
				if firstError == nil {
					firstError = result.Error
				}
				logrus.Errorf("Segment %d failed: %v", result.Index, result.Error)
				// 继续收集其他结果以避免阻塞
				continue
			}
			
			if result.Index < 0 || result.Index >= len(audioSegments) {
				return nil, fmt.Errorf("invalid segment index: %d", result.Index)
			}
			
			audioSegments[result.Index] = result.AudioData
			totalAudioSize += int64(len(result.AudioData))
			
			logrus.Debugf("Received segment %d/%d (%d bytes, took %v)",
				i+1, len(segments), len(result.AudioData), result.Duration)
		
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled during result collection: %w", ctx.Err())
		}
	}
	
	// 检查是否有错误
	if errorCount > 0 {
		return nil, fmt.Errorf("synthesis failed: %d/%d segments failed, first error: %w",
			errorCount, len(segments), firstError)
	}
	
	// 验证所有片段都已收集
	for idx, segment := range audioSegments {
		if segment == nil {
			return nil, fmt.Errorf("missing audio segment at index %d", idx)
		}
	}
	
	collectDuration := time.Since(collectStart)
	logrus.Infof("Collected %d audio segments (%d bytes total) in %v",
		len(segments), totalAudioSize, collectDuration)

	// 5. 合并音频
	mergeStart := time.Now()
	merged, err := s.merger.Merge(audioSegments)
	if err != nil {
		return nil, fmt.Errorf("failed to merge audio segments: %w", err)
	}

	// 清理音频片段,释放内存
	for i := range audioSegments {
		audioSegments[i] = nil
	}

	mergeDuration := time.Since(mergeStart)
	totalDuration := time.Since(startTime)
	
	logrus.Infof("Long text synthesis completed in %v (segment: %v, submit: %v, collect: %v, merge: %v)",
		totalDuration, segmentDuration, submitDuration, collectDuration, mergeDuration)

	// 6. 获取工作池统计
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