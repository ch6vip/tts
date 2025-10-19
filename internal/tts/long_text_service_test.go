package tts

import (
	"testing"
	"unicode/utf8"
	"tts/internal/models"
)

// TestLongTextTTSService_TextSegmentation 测试文本分段逻辑
func TestLongTextTTSService_TextSegmentation(t *testing.T) {
	// 创建长文本服务
	service := &LongTextTTSService{
		segmenter:     NewSmartSegmenter(),
		maxSegmentLen: 100,
		minTextForSplit: 500,
	}
	
	// 测试用例1: 短文本不分段
	req1 := models.TTSRequest{
		Text:  "这是一个简短的文本。",
		Voice: "zh-CN-XiaoxiaoNeural",
		Rate:  "0",
		Pitch: "0",
		Style: "general",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	segments := service.segmenter.Segment(req1.Text, service.maxSegmentLen)
	if len(segments) != 1 {
		t.Errorf("Expected 1 segment for short text, got %d", len(segments))
	}
	
	// 测试用例2: 长文本分段
	longText := "这是第一段文本。包含多个句子！用来测试分段处理？这是第二段文本。也有多个句子。继续测试功能。第三段文本。确保分段正常工作。这是第四段文本。包含更多内容。测试功能完整性。第五段文本。最后一段测试。这是第六段文本。包含更多内容。测试功能完整性。第七段文本。最后一段测试。这是第八段文本。包含更多内容。测试功能完整性。第九段文本。最后一段测试。这是第十段文本。包含更多内容。测试功能完整性。第十一段文本。最后一段测试。"
	
	segments = service.segmenter.Segment(longText, service.maxSegmentLen)
	if len(segments) <= 1 {
		t.Errorf("Expected multiple segments for long text, got %d", len(segments))
	}
	
	// 验证每个片段不超过最大长度
	for i, seg := range segments {
		segLen := utf8.RuneCountInString(seg)
		if segLen > service.maxSegmentLen {
			t.Errorf("Segment %d length %d exceeds maxLen %d", i, segLen, service.maxSegmentLen)
		}
	}
	
	t.Logf("Long text segmented into %d parts", len(segments))
}

// TestLongTextTTSService_SmartSegmentation 测试智能分段策略
func TestLongTextTTSService_SmartSegmentation(t *testing.T) {
	// 创建长文本服务，使用智能分段
	service := &LongTextTTSService{
		segmenter:     NewSmartSegmenter(),
		maxSegmentLen: 50,
		minTextForSplit: 100,
	}
	
	// 测试文本，包含多种标点符号
	text := "这是第一句话。这是第二句话！这是第三句话？这是第四句话；这是第五句话。"
	
	segments := service.segmenter.Segment(text, service.maxSegmentLen)
	
	// 验证分段结果
	if len(segments) == 0 {
		t.Error("No segments generated")
	}
	
	// 验证每个片段不超过最大长度
	for i, seg := range segments {
		segLen := utf8.RuneCountInString(seg)
		if segLen > service.maxSegmentLen {
			t.Errorf("Segment %d length %d exceeds maxLen %d", i, segLen, service.maxSegmentLen)
		}
	}
	
	// 验证拼接后的文本与原文本一致
	var reconstructed string
	for _, seg := range segments {
		reconstructed += seg
	}
	
	if reconstructed != text {
		t.Errorf("Reconstructed text doesn't match original")
		t.Logf("Original: %q", text)
		t.Logf("Reconstructed: %q", reconstructed)
	}
	
	t.Logf("Text segmented into %d parts", len(segments))
}

// TestLongTextTTSService_FixedLengthSegmentation 测试固定长度分段策略
func TestLongTextTTSService_FixedLengthSegmentation(t *testing.T) {
	// 创建长文本服务，使用固定长度分段
	service := &LongTextTTSService{
		segmenter:     NewFixedLengthSegmenter(),
		maxSegmentLen: 20,
		minTextForSplit: 50,
	}
	
	// 测试文本
	text := "这是一段测试文本，用于测试固定长度分段功能。"
	
	segments := service.segmenter.Segment(text, service.maxSegmentLen)
	
	// 验证分段结果
	if len(segments) == 0 {
		t.Error("No segments generated")
	}
	
	// 验证每个片段不超过最大长度
	for i, seg := range segments {
		segLen := utf8.RuneCountInString(seg)
		if segLen > service.maxSegmentLen {
			t.Errorf("Segment %d length %d exceeds maxLen %d", i, segLen, service.maxSegmentLen)
		}
	}
	
	// 验证拼接后的文本与原文本一致
	var reconstructed string
	for _, seg := range segments {
		reconstructed += seg
	}
	
	if reconstructed != text {
		t.Errorf("Reconstructed text doesn't match original")
		t.Logf("Original: %q", text)
		t.Logf("Reconstructed: %q", reconstructed)
	}
	
	t.Logf("Text segmented into %d parts", len(segments))
}