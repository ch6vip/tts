package tts

import (
	"context"
	"testing"
	"tts/internal/models"
)

// TestGenerateCacheKey 测试缓存键生成是否包含所有影响输出的参数
func TestGenerateCacheKey(t *testing.T) {
	// 这里我们需要一个方法来设置测试配置，但由于config包的设计，
	// 我们将通过其他方式测试
	
	// 测试用例1: Text模式，所有参数都不同
	req1 := models.TTSRequest{
		Text:  "测试文本",
		Voice: "zh-CN-XiaoxiaoNeural",
		Rate:  "0",
		Pitch: "0",
		Style: "general",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	// 测试用例2: Text模式，格式不同
	req2 := models.TTSRequest{
		Text:  "测试文本",
		Voice: "zh-CN-XiaoxiaoNeural",
		Rate:  "0",
		Pitch: "0",
		Style: "general",
		Format: "audio-16khz-128kbitrate-mono-mp3",
	}
	
	// 测试用例3: Text模式，语速不同
	req3 := models.TTSRequest{
		Text:  "测试文本",
		Voice: "zh-CN-XiaoxiaoNeural",
		Rate:  "10",
		Pitch: "0",
		Style: "general",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	// 测试用例4: SSML模式，内容相同但语音不同
	req4 := models.TTSRequest{
		SSML:  "<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='zh-CN'><voice name='zh-CN-XiaoxiaoNeural'><prosody rate='0%'>测试文本</prosody></voice></speak>",
		Voice: "zh-CN-XiaoxiaoNeural",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	// 测试用例5: SSML模式，内容相同但语音不同
	req5 := models.TTSRequest{
		SSML:  "<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='zh-CN'><voice name='zh-CN-XiaoxiaoNeural'><prosody rate='0%'>测试文本</prosody></voice></speak>",
		Voice: "zh-CN-YunxiNeural",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	// 测试用例6: SSML模式，内容相同但格式不同
	req6 := models.TTSRequest{
		SSML:  "<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='zh-CN'><voice name='zh-CN-XiaoxiaoNeural'><prosody rate='0%'>测试文本</prosody></voice></speak>",
		Voice: "zh-CN-XiaoxiaoNeural",
		Format: "audio-16khz-128kbitrate-mono-mp3",
	}
	
	// 创建缓存服务（使用模拟的下一个服务）
	cacheService := &cachingService{
		next: &mockTTSService{},
	}
	
	// 生成缓存键
	key1 := cacheService.generateCacheKey(req1)
	key2 := cacheService.generateCacheKey(req2)
	key3 := cacheService.generateCacheKey(req3)
	key4 := cacheService.generateCacheKey(req4)
	key5 := cacheService.generateCacheKey(req5)
	key6 := cacheService.generateCacheKey(req6)
	
	// 验证不同参数生成不同的缓存键
	if key1 == key2 {
		t.Error("不同的音频格式应该生成不同的缓存键")
	}
	
	if key1 == key3 {
		t.Error("不同的语速应该生成不同的缓存键")
	}
	
	if key4 == key5 {
		t.Error("SSML模式下，不同的语音应该生成不同的缓存键")
	}
	
	if key4 == key6 {
		t.Error("SSML模式下，不同的音频格式应该生成不同的缓存键")
	}
	
	// 验证相同参数生成相同的缓存键
	key1Copy := cacheService.generateCacheKey(req1)
	if key1 != key1Copy {
		t.Error("相同参数应该生成相同的缓存键")
	}
	
	key4Copy := cacheService.generateCacheKey(req4)
	if key4 != key4Copy {
		t.Error("相同参数应该生成相同的缓存键")
	}
}

// TestCacheKeyConsistency 测试缓存键的一致性，确保相同内容生成相同的键
func TestCacheKeyConsistency(t *testing.T) {
	cacheService := &cachingService{
		next: &mockTTSService{},
	}
	
	// 测试用例1: 相同内容但不同大小写和空格
	req1 := models.TTSRequest{
		Text:  "测试文本",
		Voice: "zh-CN-XiaoxiaoNeural",
		Rate:  "0",
		Pitch: "0",
		Style: "general",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	req2 := models.TTSRequest{
		Text:  "  测试文本  ",
		Voice: "  zh-CN-XiaoxiaoNeural  ",
		Rate:  " 0 ",
		Pitch: " 0 ",
		Style: " GENERAL ",
		Format: " audio-24khz-48kbitrate-mono-mp3 ",
	}
	
	key1 := cacheService.generateCacheKey(req1)
	key2 := cacheService.generateCacheKey(req2)
	
	if key1 != key2 {
		t.Error("相同内容但不同大小写和空格应该生成相同的缓存键")
	}
	
	// 测试用例2: SSML模式的一致性
	req3 := models.TTSRequest{
		SSML:  "<speak>测试文本</speak>",
		Voice: "zh-CN-XiaoxiaoNeural",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	req4 := models.TTSRequest{
		SSML:  "  <speak>测试文本</speak>  ",
		Voice: "  zh-CN-XiaoxiaoNeural  ",
		Format: " audio-24khz-48kbitrate-mono-mp3 ",
	}
	
	key3 := cacheService.generateCacheKey(req3)
	key4 := cacheService.generateCacheKey(req4)
	
	if key3 != key4 {
		t.Error("相同SSML内容但不同大小写和空格应该生成相同的缓存键")
	}
}

// TestCacheKeyUniqueness 测试不同核心参数生成不同的缓存键
func TestCacheKeyUniqueness(t *testing.T) {
	cacheService := &cachingService{
		next: &mockTTSService{},
	}
	
	baseReq := models.TTSRequest{
		Text:  "测试文本",
		Voice: "zh-CN-XiaoxiaoNeural",
		Rate:  "0",
		Pitch: "0",
		Style: "general",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	baseKey := cacheService.generateCacheKey(baseReq)
	
	// 测试不同文本
	textReq := baseReq
	textReq.Text = "不同的文本"
	if baseKey == cacheService.generateCacheKey(textReq) {
		t.Error("不同文本应该生成不同的缓存键")
	}
	
	// 测试不同语音
	voiceReq := baseReq
	voiceReq.Voice = "zh-CN-YunxiNeural"
	if baseKey == cacheService.generateCacheKey(voiceReq) {
		t.Error("不同语音应该生成不同的缓存键")
	}
	
	// 测试不同语速
	rateReq := baseReq
	rateReq.Rate = "10"
	if baseKey == cacheService.generateCacheKey(rateReq) {
		t.Error("不同语速应该生成不同的缓存键")
	}
	
	// 测试不同音调
	pitchReq := baseReq
	pitchReq.Pitch = "10"
	if baseKey == cacheService.generateCacheKey(pitchReq) {
		t.Error("不同音调应该生成不同的缓存键")
	}
	
	// 测试不同风格
	styleReq := baseReq
	styleReq.Style = "cheerful"
	if baseKey == cacheService.generateCacheKey(styleReq) {
		t.Error("不同风格应该生成不同的缓存键")
	}
	
	// 测试不同格式
	formatReq := baseReq
	formatReq.Format = "audio-16khz-128kbitrate-mono-mp3"
	if baseKey == cacheService.generateCacheKey(formatReq) {
		t.Error("不同格式应该生成不同的缓存键")
	}
}

// TestCacheKeySSMLConsistency 测试SSML模式的缓存键一致性
func TestCacheKeySSMLConsistency(t *testing.T) {
	cacheService := &cachingService{
		next: &mockTTSService{},
	}
	
	// 测试相同SSML内容但参数顺序不同
	req1 := models.TTSRequest{
		SSML:   "<speak>测试文本</speak>",
		Voice:  "zh-CN-XiaoxiaoNeural",
		Rate:   "10",
		Pitch:  "5",
		Style:  "cheerful",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	req2 := models.TTSRequest{
		Style:  "cheerful",
		Format: "audio-24khz-48kbitrate-mono-mp3",
		SSML:   "<speak>测试文本</speak>",
		Rate:   "10",
		Pitch:  "5",
		Voice:  "zh-CN-XiaoxiaoNeural",
	}
	
	key1 := cacheService.generateCacheKey(req1)
	key2 := cacheService.generateCacheKey(req2)
	
	if key1 != key2 {
		t.Error("相同SSML内容但参数顺序不同应该生成相同的缓存键")
	}
	
	// 测试SSML和Text模式的区别
	textReq := models.TTSRequest{
		Text:   "测试文本",
		Voice:  "zh-CN-XiaoxiaoNeural",
		Rate:   "10",
		Pitch:  "5",
		Style:  "cheerful",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	textKey := cacheService.generateCacheKey(textReq)
	if key1 == textKey {
		t.Error("SSML模式和Text模式应该生成不同的缓存键")
	}
}

// mockTTSService 是一个模拟的TTS服务，用于测试
type mockTTSService struct{}

func (m *mockTTSService) ListVoices(ctx context.Context, locale string) ([]models.Voice, error) {
	return nil, nil
}

func (m *mockTTSService) SynthesizeSpeech(ctx context.Context, req models.TTSRequest) (*models.TTSResponse, error) {
	return nil, nil
}