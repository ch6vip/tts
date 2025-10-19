package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

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

// Merge 使用 FFmpeg concat demuxer 合并音频片段
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
	
	// 创建临时工作目录
	workDir, err := os.MkdirTemp(m.tmpDir, "tts_merge_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)
	
	// 写入音频片段到临时文件
	var concatList bytes.Buffer
	for i, seg := range segments {
		tmpFile := filepath.Join(workDir, fmt.Sprintf("seg_%03d.mp3", i))
		if err := os.WriteFile(tmpFile, seg, 0644); err != nil {
			return nil, fmt.Errorf("failed to write segment %d: %w", i, err)
		}
		
		// 写入 concat 列表（使用相对路径避免路径问题）
		fmt.Fprintf(&concatList, "file '%s'\n", filepath.Base(tmpFile))
	}
	
	// 写入 concat 列表文件
	concatFile := filepath.Join(workDir, "concat.txt")
	if err := os.WriteFile(concatFile, concatList.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("failed to write concat file: %w", err)
	}
	
	// 执行 FFmpeg 合并
	outputFile := filepath.Join(workDir, "output.mp3")
	cmd := exec.Command(
		m.ffmpegPath,
		"-f", "concat",
		"-safe", "0",
		"-i", "concat.txt",
		"-c", "copy",      // 直接复制流，避免重新编码
		"-y",              // 覆盖输出文件
		outputFile,
	)
	cmd.Dir = workDir // 在工作目录执行命令
	
	// 捕获错误输出
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		m.logger.Error().
			Err(err).
			Str("stderr", stderr.String()).
			Msg("FFmpeg merge failed")
		// 如果 FFmpeg 失败，回退到简单合并
		return m.simpleMerge(segments)
	}
	
	// 读取合并结果
	merged, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read merged file: %w", err)
	}
	
	m.logger.Info().Int("segments", len(segments)).Msg("Successfully merged audio segments using FFmpeg")
	return merged, nil
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

// MergeToWriter 将合并结果直接写入 Writer（避免内存占用）
func (s *StreamMerger) MergeToWriter(segments [][]byte, writer io.Writer) error {
	if len(segments) == 0 {
		return errors.New("no segments to merge")
	}
	
	if len(segments) == 1 {
		_, err := writer.Write(segments[0])
		return err
	}
	
	// 创建临时目录
	workDir, err := os.MkdirTemp("", "tts_stream_*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)
	
	// 写入片段文件
	var concatList bytes.Buffer
	for i, seg := range segments {
		tmpFile := filepath.Join(workDir, fmt.Sprintf("seg_%03d.mp3", i))
		if err := os.WriteFile(tmpFile, seg, 0644); err != nil {
			return err
		}
		fmt.Fprintf(&concatList, "file '%s'\n", filepath.Base(tmpFile))
	}
	
	concatFile := filepath.Join(workDir, "concat.txt")
	if err := os.WriteFile(concatFile, concatList.Bytes(), 0644); err != nil {
		return err
	}
	
	// 使用 pipe 直接输出到 writer
	cmd := exec.Command(
		s.ffmpegPath,
		"-f", "concat",
		"-safe", "0",
		"-i", "concat.txt",
		"-c", "copy",
		"-f", "mp3",      // 输出格式
		"pipe:1",         // 输出到 stdout
	)
	cmd.Dir = workDir
	cmd.Stdout = writer
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg stream merge failed: %w, stderr: %s", err, stderr.String())
	}
	
	return nil
}