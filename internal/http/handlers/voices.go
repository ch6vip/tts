package handlers

import (
	"net/http"
	"sort"
	"strings"
	"tts/internal/tts"

	"github.com/gin-gonic/gin"
)

// VoicesHandler 处理语音列表请求
type VoicesHandler struct {
	ttsService tts.Service
}

// NewVoicesHandler 创建一个新的语音列表处理器
func NewVoicesHandler(service tts.Service) *VoicesHandler {
	return &VoicesHandler{
		ttsService: service,
	}
}

// HandleVoices 处理语音列表请求
func (h *VoicesHandler) HandleVoices(c *gin.Context) {
	// 从查询参数中获取语言筛选
	locale := c.Query("locale")

	// 获取语音列表
	voices, err := h.ttsService.ListVoices(c.Request.Context(), locale)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "获取语音列表失败: " + err.Error()})
		return
	}

	// 排序：晓晓优先，然后是中文语音，最后按本地名称排序
	sort.Slice(voices, func(i, j int) bool {
		// 检查是否是晓晓（Xiaoxiao）
		isXiaoxiaoI := strings.Contains(voices[i].ShortName, "Xiaoxiao") || strings.Contains(voices[i].LocalName, "晓晓")
		isXiaoxiaoJ := strings.Contains(voices[j].ShortName, "Xiaoxiao") || strings.Contains(voices[j].LocalName, "晓晓")

		if isXiaoxiaoI != isXiaoxiaoJ {
			return isXiaoxiaoI // 晓晓排在最前面
		}

		// 检查是否是中文语音
		isChineseI := strings.HasPrefix(voices[i].Locale, "zh")
		isChineseJ := strings.HasPrefix(voices[j].Locale, "zh")
		if isChineseI != isChineseJ {
			return isChineseI // 中文的排前面
		}

		// 同语言按本地名称排序
		return voices[i].LocalName < voices[j].LocalName
	})

	// 返回JSON响应 - 包装在对象中以匹配前端期望的格式
	c.JSON(http.StatusOK, gin.H{
		"voices": voices,
		"count":  len(voices),
	})
}
