package tts

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// SegmentationStrategy 定义文本分段策略接口
type SegmentationStrategy interface {
	Segment(text string, maxLen int) []string
}

// SmartSegmenter 智能分段器，基于句法边界进行分段
type SmartSegmenter struct {
	// 中英文句子结束标点的正则表达式
	sentenceRegex *regexp.Regexp
	// 最小段落长度（避免过小的片段）
	minSegmentLen int
}

// NewSmartSegmenter 创建智能分段器
func NewSmartSegmenter() *SmartSegmenter {
	// 匹配中英文句子结束标点：。！？；.!?;
	// 同时处理省略号等特殊情况
	return &SmartSegmenter{
		sentenceRegex: regexp.MustCompile(`([。！？；\.!\?;]+)`),
		minSegmentLen: 50, // 最小 50 字符
	}
}

// Segment 将文本智能分段
// 策略：优先在句子边界切割，采用贪心算法合并句子以最大化利用长度限制
func (s *SmartSegmenter) Segment(text string, maxLen int) []string {
	if text == "" {
		return []string{}
	}

	// 如果文本长度小于限制，直接返回
	textLen := utf8.RuneCountInString(text)
	if textLen <= maxLen {
		return []string{text}
	}

	// 1. 按段落分割（双换行符）
	paragraphs := strings.Split(text, "\n\n")
	
	var segments []string
	for _, para := range paragraphs {
		// 跳过空段落
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// 2. 对每个段落进行句子级分割
		sentences := s.splitBySentence(para)
		
		// 3. 贪心合并句子直到接近 maxLen
		merged := s.mergeSentences(sentences, maxLen)
		segments = append(segments, merged...)
	}

	// 如果没有生成任何片段（不应该发生），返回原文本
	if len(segments) == 0 {
		return []string{text}
	}

	return segments
}

// splitBySentence 按句子分割文本
func (s *SmartSegmenter) splitBySentence(text string) []string {
	if text == "" {
		return []string{}
	}

	// 使用正则表达式查找所有句子结束标点的位置
	matches := s.sentenceRegex.FindAllStringIndex(text, -1)
	
	if len(matches) == 0 {
		// 没有找到句子结束符，返回整个文本
		return []string{text}
	}

	var sentences []string
	lastEnd := 0
	
	for _, match := range matches {
		// match[0] 是标点开始位置，match[1] 是标点结束位置
		sentenceEnd := match[1]
		
		// 提取句子（包含结束标点）
		sentence := text[lastEnd:sentenceEnd]
		sentence = strings.TrimSpace(sentence)
		
		if sentence != "" {
			sentences = append(sentences, sentence)
		}
		
		lastEnd = sentenceEnd
	}
	
	// 处理最后一个句子之后的文本（如果有）
	if lastEnd < len(text) {
		remaining := strings.TrimSpace(text[lastEnd:])
		if remaining != "" {
			sentences = append(sentences, remaining)
		}
	}
	
	return sentences
}

// mergeSentences 贪心合并句子，最大化利用长度限制
func (s *SmartSegmenter) mergeSentences(sentences []string, maxLen int) []string {
	if len(sentences) == 0 {
		return []string{}
	}

	var segments []string
	var currentSegment strings.Builder
	currentLen := 0

	for _, sentence := range sentences {
		sentenceLen := utf8.RuneCountInString(sentence)
		
		// 如果单个句子超过最大长度，需要进一步切割
		if sentenceLen > maxLen {
			// 先保存当前累积的片段
			if currentSegment.Len() > 0 {
				segments = append(segments, currentSegment.String())
				currentSegment.Reset()
				currentLen = 0
			}
			
			// 对超长句子进行字符级切割
			splitLong := s.splitLongSentence(sentence, maxLen)
			segments = append(segments, splitLong...)
			continue
		}
		
		// 检查添加这个句子后是否超过限制
		testLen := currentLen + sentenceLen
		
		if currentLen > 0 && testLen > maxLen {
			// 添加会超过限制，保存当前片段
			segments = append(segments, currentSegment.String())
			currentSegment.Reset()
			currentSegment.WriteString(sentence)
			currentLen = sentenceLen
		} else {
			// 可以添加到当前片段
			currentSegment.WriteString(sentence)
			currentLen = testLen
		}
	}

	// 保存最后一个片段
	if currentSegment.Len() > 0 {
		segments = append(segments, currentSegment.String())
	}

	return segments
}

// splitLongSentence 对超长句子进行字符级切割（保底策略）
func (s *SmartSegmenter) splitLongSentence(sentence string, maxLen int) []string {
	var segments []string
	runes := []rune(sentence)
	
	for i := 0; i < len(runes); i += maxLen {
		end := i + maxLen
		if end > len(runes) {
			end = len(runes)
		}
		segments = append(segments, string(runes[i:end]))
	}
	
	return segments
}

// FixedLengthSegmenter 固定长度分段器（简单实现，用于对比）
type FixedLengthSegmenter struct{}

// NewFixedLengthSegmenter 创建固定长度分段器
func NewFixedLengthSegmenter() *FixedLengthSegmenter {
	return &FixedLengthSegmenter{}
}

// Segment 固定长度分段
func (f *FixedLengthSegmenter) Segment(text string, maxLen int) []string {
	if text == "" {
		return []string{}
	}

	runes := []rune(text)
	var segments []string
	
	for i := 0; i < len(runes); i += maxLen {
		end := i + maxLen
		if end > len(runes) {
			end = len(runes)
		}
		segments = append(segments, string(runes[i:end]))
	}
	
	return segments
}