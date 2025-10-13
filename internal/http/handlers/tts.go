package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	custom_errors "tts/internal/errors"
	"tts/internal/config"
	"tts/internal/models"
	"tts/internal/tts"
	"tts/internal/utils"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var cfg = config.Get()
// UpstreamErrorType 上游错误类型
type UpstreamErrorType int

const (
	UpstreamAuthError UpstreamErrorType = iota
	UpstreamTimeoutError
	UpstreamRateLimitError
	UpstreamInvalidRequestError
	UpstreamServerError
	UpstreamNetworkError
)

// classifyUpstreamError 分类上游服务错误
func classifyUpstreamError(err error, statusCode int) UpstreamErrorType {
	errMsg := err.Error()
	
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return UpstreamAuthError
	case http.StatusTooManyRequests:
		return UpstreamRateLimitError
	case http.StatusBadRequest:
		return UpstreamInvalidRequestError
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
		return UpstreamServerError
	}
	
	// 根据错误消息进一步分类
	errLower := strings.ToLower(errMsg)
	if strings.Contains(errLower, "timeout") || strings.Contains(errLower, "deadline exceeded") {
		return UpstreamTimeoutError
	}
	if strings.Contains(errLower, "connection") || strings.Contains(errLower, "network") {
		return UpstreamNetworkError
	}
	if strings.Contains(errLower, "unauthorized") || strings.Contains(errLower, "authentication") {
		return UpstreamAuthError
	}
	
	return UpstreamServerError
}

// getDetailedErrorMessage 根据错误类型返回详细错误信息
func getDetailedErrorMessage(errType UpstreamErrorType, originalErr error) string {
	switch errType {
	case UpstreamAuthError:
		return "上游TTS服务认证失败，请检查API密钥配置"
	case UpstreamTimeoutError:
		return "上游TTS服务响应超时，请稍后重试"
	case UpstreamRateLimitError:
		return "上游TTS服务请求频率超限，请稍后重试"
	case UpstreamInvalidRequestError:
		return fmt.Sprintf("上游TTS服务拒绝请求：%v", originalErr)
	case UpstreamServerError:
		return "上游TTS服务暂时不可用，请稍后重试"
	case UpstreamNetworkError:
		return "网络连接错误，无法访问上游TTS服务"
	default:
		return fmt.Sprintf("上游TTS服务错误：%v", originalErr)
	}
}


// getLoggerWithTraceID 从 Gin 上下文中获取带有 trace_id 的日志记录器
func getLoggerWithTraceID(c *gin.Context) *logrus.Entry {
	traceID, exists := c.Get("trace_id")
	if !exists {
		traceID = "unknown"
	}
	return logrus.WithField("trace_id", traceID)
}

// truncateForLog 截断文本用于日志显示，同时显示开头和结尾
func truncateForLog(text string, maxLength int) string {
	// 先去除换行符
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")

	runes := []rune(text)
	if len(runes) <= maxLength {
		return text
	}
	// 计算开头和结尾各显示多少字符
	halfLength := maxLength / 2
	return string(runes[:halfLength]) + "..." + string(runes[len(runes)-halfLength:])
}

// audioMerge 音频合并
func audioMerge(audioSegments [][]byte) ([]byte, error) {
	if len(audioSegments) == 0 {
		return nil, fmt.Errorf("没有音频片段可合并")
	}

	// 使用 ffmpeg 合并音频
	tempDir, err := os.MkdirTemp("", "audio_merge_")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	listFile := filepath.Join(tempDir, "concat.txt")
	lf, err := os.Create(listFile)
	if err != nil {
		return nil, err
	}

	for i, seg := range audioSegments {
		segFile := filepath.Join(tempDir, fmt.Sprintf("seg_%d.mp3", i))
		if err := os.WriteFile(segFile, seg, 0644); err != nil {
			return nil, err
		}
		if _, err := lf.WriteString(fmt.Sprintf("file '%s'\n", segFile)); err != nil {
			return nil, err
		}
	}
	lf.Close()

	outputFile := filepath.Join(tempDir, "output.mp3")

	cmd := exec.Command("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", listFile, "-c", "copy", outputFile)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	mergedData, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, err
	}
	logrus.WithField("size", formatFileSize(len(mergedData))).Info("使用ffmpeg合并完成")
	return mergedData, nil
}

// formatFileSize 格式化文件大小
func formatFileSize(size int) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.2f KB", float64(size)/1024.0)
	case size < 1024*1024*1024:
		return fmt.Sprintf("%.2f MB", float64(size)/(1024.0*1024.0))
	default:
		return fmt.Sprintf("%.2f GB", float64(size)/(1024.0*1024.0*1024.0))
	}
}

// TTSHandler 处理TTS请求
type TTSHandler struct {
	ttsService tts.Service
	config     *config.Config
}

// NewTTSHandler 创建一个新的TTS处理器
func NewTTSHandler(service tts.Service, cfg *config.Config) *TTSHandler {
	return &TTSHandler{
		ttsService: service,
		config:     cfg,
	}
}

// processTTSRequest 处理TTS请求的核心逻辑
func (h *TTSHandler) processTTSRequest(c *gin.Context, req models.TTSRequest, startTime time.Time, parseTime time.Duration, requestType string) {
	// 验证必要参数
	logger := getLoggerWithTraceID(c)
	if req.Text == "" && req.SSML == "" {
		logger.Error("错误: 未提供 text 或 ssml 参数")
		_ = c.Error(fmt.Errorf("%w: 必须提供 text 或 ssml 参数", custom_errors.ErrInvalidInput))
		return
	}

	if req.Text != "" && req.SSML != "" {
		logger.Error("错误: 不能同时提供 text 和 ssml 参数")
		_ = c.Error(fmt.Errorf("%w: 不能同时提供 text 和 ssml 参数", custom_errors.ErrInvalidInput))
		return
	}

	// 使用默认值填充空白参数
	h.fillDefaultValues(&req)

	var inputText string
	isSSML := req.SSML != ""
	if isSSML {
		inputText = req.SSML
	} else {
		inputText = req.Text
	}

	// 检查文本长度
	reqTextLength := utf8.RuneCountInString(inputText)
	if reqTextLength > h.config.TTS.MaxTextLength {
		_ = c.Error(fmt.Errorf("%w: 文本长度超过 %d 字符的限制", custom_errors.ErrInvalidInput, h.config.TTS.MaxTextLength))
		return
	}

	// 检查是否需要分段处理 (SSML不支持分段)
	segmentThreshold := h.config.TTS.SegmentThreshold
	if !isSSML && reqTextLength > segmentThreshold && reqTextLength <= h.config.TTS.MaxTextLength {
		logger.WithFields(logrus.Fields{
			"text_length": reqTextLength,
			"threshold":   segmentThreshold,
		}).Info("文本长度超过阈值，使用分段处理")
		h.handleSegmentedTTS(c, req)
		return
	}

	synthStart := time.Now()
	resp, err := h.ttsService.SynthesizeSpeech(c.Request.Context(), req)
	synthTime := time.Since(synthStart)
	logger.WithFields(logrus.Fields{
		"duration":    synthTime,
		"text_length": reqTextLength,
	}).Info("TTS合成耗时")

	if err != nil {
		// 分类错误并提供详细信息
		errType := classifyUpstreamError(err, 0)
		detailedMsg := getDetailedErrorMessage(errType, err)
		
		logger.WithFields(logrus.Fields{
			"error_type": errType,
			"original_error": err,
		}).Error("TTS合成失败")
		
		_ = c.Error(fmt.Errorf("%w: %s", custom_errors.ErrUpstreamServiceFailed, detailedMsg))
		return
	}

	// 设置响应
	c.Header("Content-Type", "audio/mpeg")
	writeStart := time.Now()
	if _, err := c.Writer.Write(resp.AudioContent); err != nil {
		logger.WithError(err).Error("写入响应失败")
		return
	}
	writeTime := time.Since(writeStart)

	// 记录总耗时
	totalTime := time.Since(startTime)
	logger.WithFields(logrus.Fields{
		"request_type": requestType,
		"total_time":   totalTime,
		"parse_time":   parseTime,
		"synth_time":   synthTime,
		"write_time":   writeTime,
		"audio_size":   formatFileSize(len(resp.AudioContent)),
	}).Info("TTS请求总耗时")
}

// fillDefaultValues 填充默认值
func (h *TTSHandler) fillDefaultValues(req *models.TTSRequest) {
	if req.Voice == "" {
		req.Voice = h.config.TTS.DefaultVoice
	}
	if req.Rate == "" {
		req.Rate = h.config.TTS.DefaultRate
	}
	if req.Pitch == "" {
		req.Pitch = h.config.TTS.DefaultPitch
	}
}

// HandleTTS 处理TTS请求
func (h *TTSHandler) HandleTTS(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodGet:
		h.HandleTTSGet(c)
	case http.MethodPost:
		h.HandleTTSPost(c)
	default:
		_ = c.Error(fmt.Errorf("%w: 仅支持GET和POST请求", custom_errors.ErrInvalidInput))
	}
}

// HandleTTSGet 处理GET方式的TTS请求
func (h *TTSHandler) HandleTTSGet(c *gin.Context) {
	startTime := time.Now()

	// 从URL参数获取
	req := models.TTSRequest{
		Text:  c.Query("t"),
		SSML:  c.Query("ssml"),
		Voice: c.Query("v"),
		Rate:  c.Query("r"),
		Pitch: c.Query("p"),
		Style: c.Query("s"),
	}

	parseTime := time.Since(startTime)
	h.processTTSRequest(c, req, startTime, parseTime, "TTS GET")
}

// HandleTTSPost 处理POST方式的TTS请求
func (h *TTSHandler) HandleTTSPost(c *gin.Context) {
	startTime := time.Now()

	// 从POST JSON体或表单数据获取
	var req models.TTSRequest
	var err error

	if c.ContentType() == "application/json" {
		err = c.ShouldBindJSON(&req)
		if err != nil {
			getLoggerWithTraceID(c).WithError(err).Error("JSON解析错误")
			_ = c.Error(fmt.Errorf("%w: 无效的JSON请求: %v", custom_errors.ErrInvalidInput, err))
			return
		}
	} else {
		err = c.ShouldBind(&req)
		if err != nil {
			getLoggerWithTraceID(c).WithError(err).Error("表单解析错误")
			_ = c.Error(fmt.Errorf("%w: 无法解析表单数据: %v", custom_errors.ErrInvalidInput, err))
			return
		}
	}

	parseTime := time.Since(startTime)
	h.processTTSRequest(c, req, startTime, parseTime, "TTS POST")
}

// HandleOpenAITTS 处理OpenAI兼容的TTS请求
func (h *TTSHandler) HandleOpenAITTS(c *gin.Context) {
	startTime := time.Now()

	// 只支持POST请求
	if c.Request.Method != http.MethodPost {
		_ = c.Error(fmt.Errorf("%w: 仅支持POST请求", custom_errors.ErrInvalidInput))
		return
	}

	// 解析请求
	var openaiReq models.OpenAIRequest
	if err := c.ShouldBindJSON(&openaiReq); err != nil {
		_ = c.Error(fmt.Errorf("%w: 无效的JSON请求: %v", custom_errors.ErrInvalidInput, err))
		return
	}

	parseTime := time.Since(startTime)

	// 检查必需字段
	if openaiReq.Input == "" {
		_ = c.Error(fmt.Errorf("%w: input字段不能为空", custom_errors.ErrInvalidInput))
		return
	}

	// 创建内部TTS请求
	req := h.convertOpenAIRequest(openaiReq)

	getLoggerWithTraceID(c).WithFields(logrus.Fields{
		"model":       openaiReq.Model,
		"from_voice":  openaiReq.Voice,
		"to_voice":    req.Voice,
		"from_speed":  openaiReq.Speed,
		"to_rate":     req.Rate,
		"text_length": utf8.RuneCountInString(req.Text),
	}).Info("OpenAI TTS请求")

	h.processTTSRequest(c, req, startTime, parseTime, "OpenAI TTS")
}

// convertOpenAIRequest 将OpenAI请求转换为内部请求格式
func (h *TTSHandler) convertOpenAIRequest(openaiReq models.OpenAIRequest) models.TTSRequest {
	// 映射OpenAI声音到Microsoft声音
	msVoice := openaiReq.Voice
	if openaiReq.Voice != "" && h.config.TTS.VoiceMapping[openaiReq.Voice] != "" {
		msVoice = h.config.TTS.VoiceMapping[openaiReq.Voice]
	}

	// 转换速度参数到微软格式
	msRate := h.config.TTS.DefaultRate
	if openaiReq.Speed != 0 {
		speedPercentage := (openaiReq.Speed - 1.0) * 100
		if speedPercentage >= 0 {
			msRate = fmt.Sprintf("+%.0f", speedPercentage)
		} else {
			msRate = fmt.Sprintf("%.0f", speedPercentage)
		}
	}

	return models.TTSRequest{
		Text:  openaiReq.Input,
		Voice: msVoice,
		Rate:  msRate,
		Pitch: h.config.TTS.DefaultPitch,
		Style: openaiReq.Model,
	}
}

// Add this struct to store synthesis results
type sentenceSynthesisResult struct {
	index     int
	length    int
	audioSize int
	content   string
	duration  time.Duration
}

// handleSegmentedTTS 处理分段TTS请求，使用流式合并减少延迟
func (h *TTSHandler) handleSegmentedTTS(c *gin.Context, req models.TTSRequest) {
	segmentStart := time.Now()
	text := req.Text
	logger := getLoggerWithTraceID(c)

	// 开始计时：分割文本
	splitStart := time.Now()
	sentences := splitTextBySentences(text)
	segmentCount := len(sentences)
	splitTime := time.Since(splitStart)

	logger.WithFields(logrus.Fields{
		"duration":      splitTime,
		"total_length":  utf8.RuneCountInString(text),
		"segment_count": segmentCount,
		"avg_sent_len":  float64(utf8.RuneCountInString(text)) / float64(segmentCount),
	}).Info("分割文本耗时")

	// 设置流式响应头
	c.Header("Content-Type", "audio/mpeg")
	c.Writer.WriteHeader(http.StatusOK)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		_ = c.Error(fmt.Errorf("%w: Streaming not supported", custom_errors.ErrUpstreamServiceFailed))
		return
	}

	// 创建有序通道用于按顺序输出音频
	type orderedAudio struct {
		index     int
		audio     []byte
		length    int
		duration  time.Duration
		content   string
	}
	
	audioChan := make(chan orderedAudio, segmentCount)
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	// 限制并发数量
	maxConcurrent := h.config.TTS.MaxConcurrent
	semaphore := make(chan struct{}, maxConcurrent)

	// 合成阶段开始时间
	synthesisStart := time.Now()

	// 并发处理每一个句子
	for i, sentence := range sentences {
		wg.Add(1)
		go func(index int, sentenceText string) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-c.Request.Context().Done():
				select {
				case errChan <- c.Request.Context().Err():
				default:
				}
				return
			}

			// 创建该句的请求
			segReq := models.TTSRequest{
				Text:  sentenceText,
				Voice: req.Voice,
				Rate:  req.Rate,
				Pitch: req.Pitch,
				Style: req.Style,
			}

			startTime := time.Now()
			resp, err := h.ttsService.SynthesizeSpeech(c.Request.Context(), segReq)
			synthDuration := time.Since(startTime)

			if err != nil {
				// 分类错误并记录
				errType := classifyUpstreamError(err, 0)
				detailedMsg := getDetailedErrorMessage(errType, err)
				
				select {
				case errChan <- fmt.Errorf("句子 %d 合成失败: %s", index+1, detailedMsg):
				default:
				}
				return
			}

			// 发送音频到通道，保持顺序
			audioChan <- orderedAudio{
				index:    index,
				audio:    resp.AudioContent,
				length:   utf8.RuneCountInString(sentenceText),
				duration: synthDuration,
				content:  truncateForLog(sentenceText, 20),
			}
		}(i, sentence)
	}

	// 等待所有goroutine完成
	go func() {
		wg.Wait()
		close(audioChan)
	}()

	// 按顺序接收并流式输出音频
	audioMap := make(map[int]orderedAudio)
	nextIndex := 0
	var totalAudioSize int
	var results []sentenceSynthesisResult

	for audio := range audioChan {
		audioMap[audio.index] = audio
		
		// 按顺序输出已准备好的音频片段
		for {
			if orderedAudio, exists := audioMap[nextIndex]; exists {
				// 立即写入响应流，实现真正的流式输出
				if _, err := c.Writer.Write(orderedAudio.audio); err != nil {
					logger.WithError(err).Error("写入响应流失败")
					return
				}
				flusher.Flush()
				
				// 记录结果
				totalAudioSize += len(orderedAudio.audio)
				results = append(results, sentenceSynthesisResult{
					index:     orderedAudio.index,
					length:    orderedAudio.length,
					audioSize: len(orderedAudio.audio),
					content:   orderedAudio.content,
					duration:  orderedAudio.duration,
				})
				
				delete(audioMap, nextIndex)
				nextIndex++
			} else {
				break
			}
		}
	}

	// 检查是否有错误发生
	select {
	case err := <-errChan:
		logger.WithError(err).Error("分段TTS处理失败")
		return
	default:
	}

	// 打印表格格式的合成结果
	logger.Info("句子合成结果表:")
	logger.Info("-------------------------------------------------------------")
	logger.Info("序号 | 长度  |    音频大小   |    耗时    | 内容")
	logger.Info("-------------------------------------------------------------")
	for _, result := range results {
		logger.Infof("#%-3d | %4d | %12s | %10v | %s",
			result.index+1,
			result.length,
			formatFileSize(result.audioSize),
			result.duration.Round(time.Millisecond),
			result.content)
	}
	logger.Info("-------------------------------------------------------------")

	// 记录总耗时
	synthesisTime := time.Since(synthesisStart)
	totalTime := time.Since(segmentStart)
	logger.WithFields(logrus.Fields{
		"total_time":   totalTime,
		"split_time":   splitTime,
		"synth_time":   synthesisTime,
		"audio_size":   formatFileSize(totalAudioSize),
		"avg_duration": synthesisTime / time.Duration(segmentCount),
	}).Info("分段流式TTS请求总耗时")
}

// HandleReader 返回 reader 可导入的格式
func (h *TTSHandler) HandleReader(context *gin.Context) {
	// 从URL参数获取
	req := models.TTSRequest{
		Text:  context.Query("t"),
		Voice: context.Query("v"),
		Rate:  context.Query("r"),
		Pitch: context.Query("p"),
		Style: context.Query("s"),
	}
	displayName := context.Query("n")

	baseUrl := utils.GetBaseURL(context)
	basePath, err := utils.JoinURL(baseUrl, cfg.Server.BasePath)
	if err != nil {
		_ = context.Error(fmt.Errorf("%w: %v", custom_errors.ErrUpstreamServiceFailed, err))
		return
	}

	// 构建基本URL
	urlParams := []string{"t={{java.encodeURI(speakText)}}", "r={{speakSpeed*4}}"}

	// 只有有值的参数才添加
	if req.Voice != "" {
		urlParams = append(urlParams, fmt.Sprintf("v=%s", req.Voice))
	}

	if req.Pitch != "" {
		urlParams = append(urlParams, fmt.Sprintf("p=%s", req.Pitch))
	}

	if req.Style != "" {
		urlParams = append(urlParams, fmt.Sprintf("s=%s", req.Style))
	}

	if cfg.TTS.ApiKey != "" {
		urlParams = append(urlParams, fmt.Sprintf("api_key=%s", cfg.TTS.ApiKey))
	}

	url := fmt.Sprintf("%s/tts?%s", basePath, strings.Join(urlParams, "&"))

	encoder := json.NewEncoder(context.Writer)
	encoder.SetEscapeHTML(false)
	context.Status(http.StatusOK)
	if err := encoder.Encode(models.ReaderResponse{
		Id:   time.Now().Unix(),
		Name: displayName,
		Url:  url,
	}); err != nil {
		getLoggerWithTraceID(context).WithError(err).Error("写入响应失败")
		_ = context.Error(fmt.Errorf("%w: 写入响应失败", custom_errors.ErrUpstreamServiceFailed))
	}
}

// HandleIFreeTime 处理IFreeTime应用请求
func (h *TTSHandler) HandleIFreeTime(context *gin.Context) {
	// 从URL参数获取
	req := models.TTSRequest{
		Voice: context.Query("v"),
		Rate:  context.Query("r"),
		Pitch: context.Query("p"),
		Style: context.Query("s"),
	}
	displayName := context.Query("n")

	// 获取基础URL
	baseUrl := utils.GetBaseURL(context)
	basePath, err := utils.JoinURL(baseUrl, cfg.Server.BasePath)
	if err != nil {
		_ = context.Error(fmt.Errorf("%w: %v", custom_errors.ErrUpstreamServiceFailed, err))
		return
	}

	// 构建URL
	url := fmt.Sprintf("%s/tts", basePath)

	// 生成随机的唯一ID
	ttsConfigID := uuid.New().String()

	// 构建声音列表
	var voiceList []models.IFreeTimeVoice

	// 构建请求参数
	params := map[string]string{
		"t": "%@", // %@ 是 IFreeTime 中的文本占位符
		"v": req.Voice,
		"r": req.Rate,
		"p": req.Pitch,
		"s": req.Style,
	}

	// 如果需要API密钥认证，添加到请求参数
	if h.config.TTS.ApiKey != "" {
		params["api_key"] = h.config.TTS.ApiKey
	}

	// 构建响应
	response := models.IFreeTimeResponse{
		LoginUrl:       "",
		MaxWordCount:   "",
		CustomRules:    map[string]interface{}{},
		TtsConfigGroup: "Azure",
		TTSName:        displayName,
		ClassName:      "JxdAdvCustomTTS",
		TTSConfigID:    ttsConfigID,
		HttpConfigs: models.IFreeTimeHttpConfig{
			UseCookies: 1,
			Headers:    map[string]interface{}{},
		},
		VoiceList: voiceList,
		TtsHandles: []models.IFreeTimeTtsHandle{
			{
				ParamsEx:         "",
				ProcessType:      1,
				MaxPageCount:     1,
				NextPageMethod:   1,
				Method:           1,
				RequestByWebView: 0,
				Parser:           map[string]interface{}{},
				NextPageParams:   map[string]interface{}{},
				Url:              url,
				Params:           params,
				HttpConfigs: models.IFreeTimeHttpConfig{
					UseCookies: 1,
					Headers:    map[string]interface{}{},
				},
			},
		},
	}

	// 设置响应类型
	context.Header("Content-Type", "application/json")
	context.JSON(http.StatusOK, response)
}

// splitTextBySentences 将文本按句子分割
func splitTextBySentences(text string) []string {
	// 如果文本过短，直接作为一个句子返回
	if utf8.RuneCountInString(text) < 100 {
		return []string{text}
	}

	maxLen := cfg.TTS.MaxSentenceLength
	minLen := cfg.TTS.MinSentenceLength

	// 第一次分割：按标点和长度限制分割
	sentences := utils.SplitAndFilterEmptyLines(text)
	// 第二次处理：合并过短的句子
	shortSentences := utils.MergeStringsWithLimit(sentences, minLen, maxLen)
	logrus.WithFields(logrus.Fields{
		"before": len(sentences),
		"after":  len(shortSentences),
	}).Info("分割后的句子数")
	return shortSentences
}
