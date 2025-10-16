package tts

import (
	"testing"
	"unicode/utf8"
)

func TestSmartSegmenter_Segment(t *testing.T) {
	segmenter := NewSmartSegmenter()

	tests := []struct {
		name        string
		text        string
		maxLen      int
		wantSegments int
		checkContent bool
	}{
		{
			name:         "短文本不分段",
			text:         "这是一个简短的文本。",
			maxLen:       500,
			wantSegments: 1,
			checkContent: false,
		},
		{
			name:         "空文本",
			text:         "",
			maxLen:       500,
			wantSegments: 0,
			checkContent: false,
		},
		{
			name:         "中文句子分段",
			text:         "这是第一句话。这是第二句话！这是第三句话？",
			maxLen:       15,
			wantSegments: 2, // 贪心合并会将前两句合并
			checkContent: true,
		},
		{
			name:         "段落分割",
			text:         "第一段内容。\n\n第二段内容。",
			maxLen:       500,
			wantSegments: 1, // 两段都很短，会合并
			checkContent: false,
		},
		{
			name:         "混合标点",
			text:         "问题一？问题二！陈述句。分号句；",
			maxLen:       10,
			wantSegments: 2, // 实际会合并成更少的段
			checkContent: false,
		},
		{
			name:         "超长句子",
			text:         "这是一个非常非常非常非常非常非常非常非常非常非常非常非常长的句子没有标点符号",
			maxLen:       20,
			wantSegments: 2, // 字符级切割
			checkContent: false,
		},
		{
			name:         "真实长文本测试",
			text:         "人工智能是计算机科学的一个分支。它企图了解智能的实质。并生产出一种新的能以人类智能相似的方式做出反应的智能机器。该领域的研究包括机器人、语言识别、图像识别、自然语言处理和专家系统等。",
			maxLen:       50,
			wantSegments: -1, // 不限制具体数量，只检查合理性
			checkContent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := segmenter.Segment(tt.text, tt.maxLen)
			
			// 如果 wantSegments 为 -1，只检查是否有合理的分段
			if tt.wantSegments == -1 {
				if len(segments) == 0 {
					t.Errorf("Segment() got 0 segments for non-empty text")
				}
			} else if len(segments) != tt.wantSegments {
				t.Errorf("Segment() got %d segments, want %d", len(segments), tt.wantSegments)
				t.Logf("Segments: %v", segments)
			}

			// 验证每个片段不超过最大长度
			for i, seg := range segments {
				segLen := utf8.RuneCountInString(seg)
				if segLen > tt.maxLen {
					t.Errorf("Segment %d length %d exceeds maxLen %d: %s", i, segLen, tt.maxLen, seg)
				}
			}

			// 验证拼接后的文本与原文本一致（去除空白）
			var reconstructed string
			for _, seg := range segments {
				reconstructed += seg
			}
			
			// 移除空白字符进行比较
			if removeWhitespace(reconstructed) != removeWhitespace(tt.text) {
				t.Errorf("Reconstructed text doesn't match original")
				t.Logf("Original: %q", tt.text)
				t.Logf("Reconstructed: %q", reconstructed)
			}
		})
	}
}

func TestSmartSegmenter_SplitBySentence(t *testing.T) {
	segmenter := NewSmartSegmenter()

	tests := []struct {
		name          string
		text          string
		wantSentences int
	}{
		{
			name:          "单句",
			text:          "这是一句话。",
			wantSentences: 1,
		},
		{
			name:          "多句",
			text:          "第一句。第二句！第三句？",
			wantSentences: 3,
		},
		{
			name:          "没有标点",
			text:          "没有标点符号的文本",
			wantSentences: 1,
		},
		{
			name:          "空文本",
			text:          "",
			wantSentences: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := segmenter.splitBySentence(tt.text)
			if len(sentences) != tt.wantSentences {
				t.Errorf("splitBySentence() got %d sentences, want %d", len(sentences), tt.wantSentences)
				t.Logf("Sentences: %v", sentences)
			}
		})
	}
}

func TestFixedLengthSegmenter_Segment(t *testing.T) {
	segmenter := NewFixedLengthSegmenter()

	tests := []struct {
		name         string
		text         string
		maxLen       int
		wantSegments int
	}{
		{
			name:         "简单分段",
			text:         "12345678901234567890",
			maxLen:       10,
			wantSegments: 2,
		},
		{
			name:         "不整除",
			text:         "123456789012345",
			maxLen:       10,
			wantSegments: 2,
		},
		{
			name:         "短文本",
			text:         "123",
			maxLen:       10,
			wantSegments: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := segmenter.Segment(tt.text, tt.maxLen)
			if len(segments) != tt.wantSegments {
				t.Errorf("Segment() got %d segments, want %d", len(segments), tt.wantSegments)
			}

			// 验证拼接后与原文本一致
			var reconstructed string
			for _, seg := range segments {
				reconstructed += seg
			}
			if reconstructed != tt.text {
				t.Errorf("Reconstructed text doesn't match original")
			}
		})
	}
}

func TestSmartSegmenter_MergeSentences(t *testing.T) {
	segmenter := NewSmartSegmenter()

	tests := []struct {
		name         string
		sentences    []string
		maxLen       int
		wantSegments int
	}{
		{
			name:         "合并短句",
			sentences:    []string{"短句一。", "短句二。", "短句三。"},
			maxLen:       20,
			wantSegments: 1, // 三个短句都能合并到20字符内
		},
		{
			name:         "单句超长",
			sentences:    []string{"这是一个非常长的句子超过了最大长度限制"},
			maxLen:       10,
			wantSegments: 2, // 会被进一步切割
		},
		{
			name:         "空列表",
			sentences:    []string{},
			maxLen:       100,
			wantSegments: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := segmenter.mergeSentences(tt.sentences, tt.maxLen)
			if len(segments) != tt.wantSegments {
				t.Errorf("mergeSentences() got %d segments, want %d", len(segments), tt.wantSegments)
				t.Logf("Segments: %v", segments)
			}
		})
	}
}

// 辅助函数：移除文本中的空白字符
func removeWhitespace(s string) string {
	result := ""
	for _, r := range s {
		if r != ' ' && r != '\n' && r != '\r' && r != '\t' {
			result += string(r)
		}
	}
	return result
}

// 基准测试
func BenchmarkSmartSegmenter_Segment(b *testing.B) {
	segmenter := NewSmartSegmenter()
	text := `这是一段测试文本。它包含多个句子！用来测试分段性能？
	这是第二段。包含更多内容。用于性能测试。
	第三段继续测试。确保算法高效。这很重要！`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = segmenter.Segment(text, 50)
	}
}

func BenchmarkFixedLengthSegmenter_Segment(b *testing.B) {
	segmenter := NewFixedLengthSegmenter()
	text := `这是一段测试文本。它包含多个句子！用来测试分段性能？
	这是第二段。包含更多内容。用于性能测试。
	第三段继续测试。确保算法高效。这很重要！`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = segmenter.Segment(text, 50)
	}
}