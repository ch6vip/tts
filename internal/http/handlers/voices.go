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

	// 排序：中文语音优先
	sort.Slice(voices, func(i, j int) bool {
		isChineseI := strings.HasPrefix(voices[i].Locale, "zh")
		isChineseJ := strings.HasPrefix(voices[j].Locale, "zh")
		if isChineseI != isChineseJ {
			return isChineseI // 中文的排前面
		}
		return voices[i].LocalName < voices[j].LocalName // 否则按本地名称排序
	})

	// 返回JSON响应
	c.JSON(http.StatusOK, voices)
}
