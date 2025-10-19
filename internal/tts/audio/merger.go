package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/rs/zerolog"
)

// Merger 音频合并器接口
type Merger interface {
	Merge(segments [][]byte) ([]byte, error)
}

// FFmpegMerger 使用 FFmpeg 进行专业音频合并
type FFmpegMerger struct {
	ffmpegPath string
	tmpDir     string
	logger     zerolog.Logger
}

// NewFFmpegMerger 创建 FFmpeg 合并器
func NewFFmpegMerger(ffmpegPath string, logger zerolog.Logger) *FFmpegMerger {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // 使用 PATH 中的 ffmpeg
	}
	
	return &FFmpegMerger{
		ffmpegPath: ffmpegPath,
		tmpDir:     os.TempDir(),
		logger:     logger,
	}
}

// Merge 使用 FFmpeg concat protocol 通过管道合并音频片段（零磁盘 I/O）
func (m *FFmpegMerger) Merge(segments [][]byte) ([]byte, error) {
	if len(segments) == 0 {
		return nil, errors.New("no segments to merge")
	}
	
	// 如果只有一个片段，直接返回
	if len(segments) == 1 {
		return segments[0], nil
	}
	
	// 检查 ffmpeg 是否可用
	if err := m.checkFFmpeg(); err != nil {
		m.logger.Warn().Err(err).Msg("FFmpeg not available, falling back to simple merge")
		return m.simpleMerge(segments)
	}
	
	// 使用管道方式合并：通过多个 FFmpeg 进程链式处理
	// 策略：使用 concat filter 而非 concat demuxer，完全通过管道传输
	return m.mergeWithPipe(segments)
}

// mergeWithPipe 使用管道方式合并音频，避免磁盘 I/O
func (m *FFmpegMerger) mergeWithPipe(segments [][]byte) ([]byte, error) {
	// 对于两个片段的情况，使用 concat filter
	if len(segments) == 2 {
		return m.mergeTwoSegments(segments[0], segments[1])
	}
	
	// 对于多个片段，递归两两合并
	// 这种方法虽然需要多次调用 FFmpeg，但完全避免了磁盘 I/O
	mid := len(segments) / 2
	
	// 并行合并左右两部分
	leftMerged, err := m.mergeWithPipe(segments[:mid])
	if err != nil {
		m.logger.Warn().Err(err).Msg("Left merge failed, falling back to simple merge")
		return m.simpleMerge(segments)
	}
	
	rightMerged, err := m.mergeWithPipe(segments[mid:])
	if err != nil {
		m.logger.Warn().Err(err).Msg("Right merge failed, falling back to simple merge")
		return m.simpleMerge(segments)
	}
	
	// 合并两个已合并的部分
	return m.mergeTwoSegments(leftMerged, rightMerged)
}

// mergeTwoSegments 使用管道合并两个音频片段
func (m *FFmpegMerger) mergeTwoSegments(seg1, seg2 []byte) ([]byte, error) {
	// 使用 FFmpeg concat filter 通过管道合并
	// 命令：ffmpeg -i pipe:0 -i pipe:1 -filter_complex "[0:a][1:a]concat=n=2:v=0:a=1" -f mp3 pipe:1
	cmd := exec.Command(
		m.ffmpegPath,
		"-i", "pipe:0",           // 第一个输入从 stdin
		"-f", "mp3",
		"-i", "pipe:3",           // 第二个输入从 fd 3
		"-filter_complex", "[0:a][1:a]concat=n=2:v=0:a=1[out]",
		"-map", "[out]",
		"-f", "mp3",
		"-c:a", "libmp3lame",     // 使用 MP3 编码器
		"-b:a", "128k",           // 设置比特率
		"pipe:1",                 // 输出到 stdout
	)
	
	// 创建输入管道
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	
	// 为第二个输入创建额外的管道
	extraPipe, extraWriter, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create extra pipe: %w", err)
	}
	defer extraPipe.Close()
	
	// 设置文件描述符 3 为第二个输入
	cmd.ExtraFiles = []*os.File{extraPipe}
	
	// 创建输出缓冲区
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// 启动 FFmpeg 进程
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}
	
	// 在 goroutine 中写入第一个片段到 stdin
	errChan1 := make(chan error, 1)
	go func() {
		defer stdin.Close()
		_, err := stdin.Write(seg1)
		errChan1 <- err
	}()
	
	// 在另一个 goroutine 中写入第二个片段到 fd 3
	errChan2 := make(chan error, 1)
	go func() {
		defer extraWriter.Close()
		_, err := extraWriter.Write(seg2)
		errChan2 <- err
	}()
	
	// 等待写入完成
	if err := <-errChan1; err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to write first segment: %w", err)
	}
	
	if err := <-errChan2; err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("failed to write second segment: %w", err)
	}
	
	// 等待 FFmpeg 完成
	if err := cmd.Wait(); err != nil {
		m.logger.Error().
			Err(err).
			Str("stderr", stderr.String()).
			Msg("FFmpeg pipe merge failed")
		return nil, fmt.Errorf("ffmpeg merge failed: %w, stderr: %s", err, stderr.String())
	}
	
	m.logger.Debug().Msg("Successfully merged two audio segments via pipe")
	return stdout.Bytes(), nil
}

// checkFFmpeg 检查 FFmpeg 是否可用
func (m *FFmpegMerger) checkFFmpeg() error {
	cmd := exec.Command(m.ffmpegPath, "-version")
	return cmd.Run()
}

// simpleMerge 简单合并（回退方案）
// 注意：此方法会移除 ID3 标签但不处理帧边界，音质可能受影响
func (m *FFmpegMerger) simpleMerge(segments [][]byte) ([]byte, error) {
	m.logger.Warn().Msg("Using simple merge (may cause audio artifacts)")
	
	merger := &SimpleMerger{logger: m.logger}
	return merger.Merge(segments)
}

// SimpleMerger 简单的字节级合并器（移除 ID3 标签）
type SimpleMerger struct {
	logger zerolog.Logger
}

// NewSimpleMerger 创建简单合并器
func NewSimpleMerger(logger zerolog.Logger) *SimpleMerger {
	return &SimpleMerger{logger: logger}
}

// Merge 执行简单的字节拼接（移除 ID3 标签）
func (s *SimpleMerger) Merge(segments [][]byte) ([]byte, error) {
	if len(segments) == 0 {
		return nil, errors.New("no segments to merge")
	}
	
	if len(segments) == 1 {
		return segments[0], nil
	}
	
	var merged bytes.Buffer
	
	for i, seg := range segments {
		// 移除 ID3 标签
		cleaned := removeID3Tags(seg)
		
		// 第一个片段保留所有数据
		if i == 0 {
			merged.Write(cleaned)
			continue
		}
		
		// 后续片段尝试跳过可能的无声帧（简化处理）
		// 跳过前 512 字节（大约 1-2 个 MP3 帧）
		skipBytes := 512
		if len(cleaned) > skipBytes {
			merged.Write(cleaned[skipBytes:])
		} else {
			merged.Write(cleaned)
		}
	}
	
	s.logger.Warn().Int("segments", len(segments)).Msg("Simple merge completed (may have audio artifacts)")
	return merged.Bytes(), nil
}

// removeID3Tags 移除 MP3 的 ID3v2 标签
func removeID3Tags(data []byte) []byte {
	if len(data) < 10 {
		return data
	}
	
	// 检查 ID3v2 标签头 "ID3"
	if data[0] == 'I' && data[1] == 'D' && data[2] == '3' {
		// ID3v2 标签大小在字节 6-9（使用同步安全整数编码）
		size := int(data[6])<<21 | int(data[7])<<14 | int(data[8])<<7 | int(data[9])
		tagSize := size + 10 // 加上 10 字节的头部
		
		if tagSize < len(data) {
			// 这个函数不在结构体方法中，无法使用 logger
			// 如果需要日志，应该将此函数改为方法或传入 logger
			return data[tagSize:]
		}
	}
	
	return data
}

// StreamMerger 流式合并器（用于大文件）
type StreamMerger struct {
	ffmpegPath string
	logger     zerolog.Logger
}

// NewStreamMerger 创建流式合并器
func NewStreamMerger(ffmpegPath string, logger zerolog.Logger) *StreamMerger {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &StreamMerger{
		ffmpegPath: ffmpegPath,
		logger:     logger,
	}
}

// MergeToWriter 将合并结果直接写入 Writer（零磁盘 I/O，避免内存占用）
func (s *StreamMerger) MergeToWriter(segments [][]byte, writer io.Writer) error {
	if len(segments) == 0 {
		return errors.New("no segments to merge")
	}
	
	if len(segments) == 1 {
		_, err := writer.Write(segments[0])
		return err
	}
	
	// 使用管道方式递归合并
	return s.mergeToWriterWithPipe(segments, writer)
}

// mergeToWriterWithPipe 使用管道递归合并音频到 Writer
func (s *StreamMerger) mergeToWriterWithPipe(segments [][]byte, writer io.Writer) error {
	// 对于两个片段的情况，直接合并到 writer
	if len(segments) == 2 {
		return s.mergeTwoSegmentsToWriter(segments[0], segments[1], writer)
	}
	
	// 对于多个片段，先合并成中间结果，再写入 writer
	if len(segments) > 2 {
		// 递归合并左右两部分到内存
		mid := len(segments) / 2
		
		var leftBuf bytes.Buffer
		if err := s.mergeToWriterWithPipe(segments[:mid], &leftBuf); err != nil {
			return err
		}
		
		var rightBuf bytes.Buffer
		if err := s.mergeToWriterWithPipe(segments[mid:], &rightBuf); err != nil {
			return err
		}
		
		// 合并两个中间结果到最终 writer
		return s.mergeTwoSegmentsToWriter(leftBuf.Bytes(), rightBuf.Bytes(), writer)
	}
	
	return errors.New("invalid segment count")
}

// mergeTwoSegmentsToWriter 使用管道合并两个音频片段到 Writer
func (s *StreamMerger) mergeTwoSegmentsToWriter(seg1, seg2 []byte, writer io.Writer) error {
	// 使用 FFmpeg concat filter 通过管道合并
	cmd := exec.Command(
		s.ffmpegPath,
		"-i", "pipe:0",           // 第一个输入从 stdin
		"-f", "mp3",
		"-i", "pipe:3",           // 第二个输入从 fd 3
		"-filter_complex", "[0:a][1:a]concat=n=2:v=0:a=1[out]",
		"-map", "[out]",
		"-f", "mp3",
		"-c:a", "libmp3lame",     // 使用 MP3 编码器
		"-b:a", "128k",           // 设置比特率
		"pipe:1",                 // 输出到 stdout
	)
	
	// 创建输入管道
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	
	// 为第二个输入创建额外的管道
	extraPipe, extraWriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create extra pipe: %w", err)
	}
	defer extraPipe.Close()
	
	// 设置文件描述符 3 为第二个输入
	cmd.ExtraFiles = []*os.File{extraPipe}
	
	// 设置输出到传入的 writer
	cmd.Stdout = writer
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	// 启动 FFmpeg 进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}
	
	// 在 goroutine 中写入第一个片段到 stdin
	errChan1 := make(chan error, 1)
	go func() {
		defer stdin.Close()
		_, err := stdin.Write(seg1)
		errChan1 <- err
	}()
	
	// 在另一个 goroutine 中写入第二个片段到 fd 3
	errChan2 := make(chan error, 1)
	go func() {
		defer extraWriter.Close()
		_, err := extraWriter.Write(seg2)
		errChan2 <- err
	}()
	
	// 等待写入完成
	if err := <-errChan1; err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to write first segment: %w", err)
	}
	
	if err := <-errChan2; err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to write second segment: %w", err)
	}
	
	// 等待 FFmpeg 完成
	if err := cmd.Wait(); err != nil {
		s.logger.Error().
			Err(err).
			Str("stderr", stderr.String()).
			Msg("FFmpeg stream pipe merge failed")
		return fmt.Errorf("ffmpeg stream merge failed: %w, stderr: %s", err, stderr.String())
	}
	
	s.logger.Debug().Msg("Successfully merged two audio segments to writer via pipe")
	return nil
}