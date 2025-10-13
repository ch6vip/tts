package handlers

import (
	"context"
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
	"tts/internal/jobs"
	"tts/internal/utils"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var cfg = config.Get()

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
	jobStore   *jobs.JobStore
}

// NewTTSHandler 创建一个新的TTS处理器
func NewTTSHandler(service tts.Service, cfg *config.Config, jobStore *jobs.JobStore) *TTSHandler {
	return &TTSHandler{
		ttsService: service,
		config:     cfg,
		jobStore:   jobStore,
	}
}

// processTTSRequest 处理TTS请求的核心逻辑
func (h *TTSHandler) processTTSRequest(c *gin.Context, req models.TTSRequest, startTime time.Time, parseTime time.Duration, requestType string) {
	logger := getLoggerWithTraceID(c)

	// 验证和填充默认值
	if err := h.validateAndFillTTSRequest(&req, logger); err != nil {
		_ = c.Error(err)
		return
	}

	inputText, isSSML := h.getInputText(req)
	reqTextLength := utf8.RuneCountInString(inputText)

	// 决定是同步还是异步处理
	// 如果是SSML或文本长度小于分段阈值，则同步处理
	if isSSML || reqTextLength <= h.config.TTS.SegmentThreshold {
		h.handleSyncTTS(c, req, startTime, parseTime, requestType, logger)
	} else {
		// 异步处理长文本
		h.handleAsyncTTS(c, req, logger)
	}
}

// validateAndFillTTSRequest 验证请求并填充默认值
func (h *TTSHandler) validateAndFillTTSRequest(req *models.TTSRequest, logger *logrus.Entry) error {
	if req.Text == "" && req.SSML == "" {
		logger.Error("错误: 未提供 text 或 ssml 参数")
		return fmt.Errorf("%w: 必须提供 text 或 ssml 参数", custom_errors.ErrInvalidInput)
	}

	if req.Text != "" && req.SSML != "" {
		logger.Error("错误: 不能同时提供 text 和 ssml 参数")
		return fmt.Errorf("%w: 不能同时提供 text 和 ssml 参数", custom_errors.ErrInvalidInput)
	}

	h.fillDefaultValues(req)

	inputText, _ := h.getInputText(*req)
	reqTextLength := utf8.RuneCountInString(inputText)
	if reqTextLength > h.config.TTS.MaxTextLength {
		return fmt.Errorf("%w: 文本长度超过 %d 字符的限制", custom_errors.ErrInvalidInput, h.config.TTS.MaxTextLength)
	}

	return nil
}

// getInputText 从请求中获取输入文本和类型
func (h *TTSHandler) getInputText(req models.TTSRequest) (text string, isSSML bool) {
	isSSML = req.SSML != ""
	if isSSML {
		text = req.SSML
	} else {
		text = req.Text
	}
	return text, isSSML
}

// handleSyncTTS 同步处理TTS请求
func (h *TTSHandler) handleSyncTTS(c *gin.Context, req models.TTSRequest, startTime time.Time, parseTime time.Duration, requestType string, logger *logrus.Entry) {
	inputText, _ := h.getInputText(req)
	reqTextLength := utf8.RuneCountInString(inputText)

	synthStart := time.Now()
	resp, err := h.ttsService.SynthesizeSpeech(c.Request.Context(), req)
	synthTime := time.Since(synthStart)

	logger.WithFields(logrus.Fields{
		"duration":    synthTime,
		"text_length": reqTextLength,
	}).Info("TTS同步合成耗时")

	if err != nil {
		logger.WithError(err).Error("TTS合成失败")
		_ = c.Error(fmt.Errorf("%w: %v", custom_errors.ErrUpstreamServiceFailed, err))
		return
	}

	c.Header("Content-Type", "audio/mpeg")
	writeStart := time.Now()
	if _, err := c.Writer.Write(resp.AudioContent); err != nil {
		logger.WithError(err).Error("写入响应失败")
		return
	}
	writeTime := time.Since(writeStart)

	totalTime := time.Since(startTime)
	logger.WithFields(logrus.Fields{
		"request_type": requestType,
		"total_time":   totalTime,
		"parse_time":   parseTime,
		"synth_time":   synthTime,
		"write_time":   writeTime,
		"audio_size":   formatFileSize(len(resp.AudioContent)),
	}).Info("TTS同步请求总耗时")
}

// handleAsyncTTS 异步处理TTS请求
func (h *TTSHandler) handleAsyncTTS(c *gin.Context, req models.TTSRequest, logger *logrus.Entry) {
	job := h.jobStore.CreateJob()
	logger.WithField("job_id", job.ID).Info("创建异步TTS任务")

	// 在后台goroutine中处理
	go h.runSynthesisJob(req, job.ID, logger)

	c.JSON(http.StatusAccepted, gin.H{
		"status":   "processing",
		"job_id":   job.ID,
		"progress": "0/0",
	})
}

// runSynthesisJob 在后台运行合成任务
func (h *TTSHandler) runSynthesisJob(req models.TTSRequest, jobID string, logger *logrus.Entry) {
	// 使用分段逻辑，但不是流式返回，而是合并后存储
	text := req.Text
	sentences := splitTextBySentences(text)
	segmentCount := len(sentences)
	logger.WithFields(logrus.Fields{
		"job_id":        jobID,
		"segment_count": segmentCount,
	}).Info("开始异步分段合成")

	var audioSegments [][]byte
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	maxConcurrent := h.config.TTS.MaxConcurrent
	semaphore := make(chan struct{}, maxConcurrent)

	for i, sentence := range sentences {
		wg.Add(1)
		go func(index int, sentenceText string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			segReq := req
			segReq.Text = sentenceText

			resp, err := h.ttsService.SynthesizeSpeech(context.Background(), segReq)
			if err != nil {
				select {
				case errChan <- fmt.Errorf("句子 %d 合成失败: %w", index+1, err):
				default:
				}
				return
			}

			mu.Lock()
			// 保持顺序
			if len(audioSegments) == index {
				audioSegments = append(audioSegments, resp.AudioContent)
			} else {
				// 如果出现乱序，需要更复杂的逻辑来保证顺序
				// 这里简化处理，假设大部分情况是顺序的
				// 实际生产中可能需要一个带索引的map来排序
				temp := make([][]byte, segmentCount)
				copy(temp, audioSegments)
				temp[index] = resp.AudioContent
				audioSegments = temp
			}
			progress := fmt.Sprintf("%d/%d", len(audioSegments), segmentCount)
			h.jobStore.UpdateProgress(jobID, progress)
			mu.Unlock()

		}(i, sentence)
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		logger.WithError(err).WithField("job_id", jobID).Error("异步TTS任务失败")
		h.jobStore.SetJobError(jobID, err.Error())
		return
	}

	logger.WithField("job_id", jobID).Info("所有分段合成完成，开始合并")
	mergedAudio, err := audioMerge(audioSegments)
	if err != nil {
		logger.WithError(err).WithField("job_id", jobID).Error("音频合并失败")
		h.jobStore.SetJobError(jobID, "音频合并失败")
		return
	}

	h.jobStore.SetJobComplete(jobID, mergedAudio)
	logger.WithFields(logrus.Fields{
		"job_id":     jobID,
		"audio_size": formatFileSize(len(mergedAudio)),
	}).Info("异步TTS任务成功完成")
}

// HandleJobStatus handles requests for job status.
func (h *TTSHandler) HandleJobStatus(c *gin.Context) {
	jobID := c.Param("job_id")
	logger := getLoggerWithTraceID(c).WithField("job_id", jobID)

	job, found := h.jobStore.GetJob(jobID)
	if !found {
		logger.Warn("查询的Job ID不存在")
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	logger.WithFields(logrus.Fields{
		"status":   job.Status,
		"progress": job.Progress,
	}).Info("查询Job状态")

	c.JSON(http.StatusOK, gin.H{
		"job_id":   job.ID,
		"status":   job.Status,
		"progress": job.Progress,
		"error":    job.Error,
	})
}

// HandleJobResult handles requests for the result of a completed job.
func (h *TTSHandler) HandleJobResult(c *gin.Context) {
	jobID := c.Param("job_id")
	logger := getLoggerWithTraceID(c).WithField("job_id", jobID)

	job, found := h.jobStore.GetJob(jobID)
	if !found {
		logger.Warn("查询的Job ID不存在")
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	if job.Status != models.JobStatusComplete {
		logger.Warn("请求Job结果，但任务尚未完成")
		c.JSON(http.StatusAccepted, gin.H{"status": job.Status, "error": "Job not complete"})
		return
	}

	logger.WithField("audio_size", formatFileSize(len(job.AudioData))).Info("提供Job结果")
	c.Data(http.StatusOK, "audio/mpeg", job.AudioData)
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
