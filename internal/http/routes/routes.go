package routes

import (
	"io/fs"
	"net/http"
	"tts/internal/config"
	"tts/internal/http/handlers"
	"tts/internal/http/middleware"
	"tts/internal/tts"
	"tts/internal/tts/microsoft"
	"tts/web"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// SetupRoutes 配置所有API路由
func SetupRoutes(cfg *config.Config, ttsService tts.Service, logger zerolog.Logger) (*gin.Engine, error) {
	// 创建Gin路由
	router := gin.New()

	// 获取底层的 microsoft.Client 用于长文本服务
	var msClient *microsoft.Client
	
	// 如果是缓存服务,需要获取底层服务
	type underlyingServiceGetter interface {
		GetUnderlyingService() tts.Service
	}
	
	if cacheSvc, ok := ttsService.(underlyingServiceGetter); ok {
		// 从缓存层获取底层服务
		msClient = cacheSvc.GetUnderlyingService().(*microsoft.Client)
	} else {
		// 直接使用原始服务
		msClient = ttsService.(*microsoft.Client)
	}

	// 创建长文本 TTS 服务
	longTextService := tts.NewLongTextTTSService(
		msClient,
		tts.LongTextConfig{
			MaxSegmentLength: cfg.TTS.LongText.MaxSegmentLength,
			WorkerCount:      cfg.TTS.LongText.WorkerCount,
			MinTextForSplit:  cfg.TTS.LongText.MinTextForSplit,
			FFmpegPath:       cfg.TTS.LongText.FFmpegPath,
			UseSmartSegment:  cfg.TTS.LongText.UseSmartSegment,
		},
		logger,
	)

	// 创建处理器
	ttsHandler := handlers.NewTTSHandler(ttsService, longTextService, cfg, logger)
	voicesHandler := handlers.NewVoicesHandler(ttsService)
	metricsHandler := handlers.NewMetricsHandler()

	// 创建页面处理器
	pagesHandler, err := handlers.NewPagesHandler(cfg)
	if err != nil {
		return nil, err
	}

	// 应用中间件
	router.Use(middleware.Logger()) // 日志中间件
	router.Use(middleware.CORS())      // CORS中间件
	router.Use(middleware.ErrorHandler(logger)) // 错误处理中间件

	// 应用基础路径前缀
	var baseRouter *gin.RouterGroup
	if cfg.Server.BasePath != "" {
		baseRouter = router.Group(cfg.Server.BasePath)
	} else {
		baseRouter = router.Group("")
	}

	// 设置静态文件服务
	// 设置静态文件服务
	staticRoot, err := fs.Sub(web.StaticFS, "static")
	if err != nil {
		return nil, err
	}
	baseRouter.StaticFS("/static", http.FS(staticRoot))

	// 设置主页路由
	baseRouter.GET("/", pagesHandler.HandleIndex)

	// 设置API文档路由
	baseRouter.GET("/api-doc", pagesHandler.HandleAPIDoc)

	// 创建 API 路由组
	apiGroup := baseRouter.Group("/api")

	// 设置TTS API路由 - 添加认证中间件
	apiGroup.POST("/tts", middleware.TTSAuth(cfg.TTS.ApiKey), ttsHandler.HandleTTS)
	apiGroup.GET("/tts", middleware.TTSAuth(cfg.TTS.ApiKey), ttsHandler.HandleTTS)

	// 设置语音列表API路由
	apiGroup.GET("/voices", voicesHandler.HandleVoices)

	// 保持旧的路由以兼容现有客户端
	baseRouter.POST("/tts", middleware.TTSAuth(cfg.TTS.ApiKey), ttsHandler.HandleTTS)
	baseRouter.GET("/tts", middleware.TTSAuth(cfg.TTS.ApiKey), ttsHandler.HandleTTS)
	baseRouter.GET("/reader.json", middleware.TTSAuth(cfg.TTS.ApiKey), ttsHandler.HandleReader)
	baseRouter.GET("ifreetime.json", middleware.TTSAuth(cfg.TTS.ApiKey), ttsHandler.HandleIFreeTime)
	baseRouter.GET("/voices", voicesHandler.HandleVoices)

	// 设置OpenAI兼容接口的处理器，添加验证中间件
	openAIHandler := middleware.OpenAIAuth(cfg.OpenAI.ApiKey)
	baseRouter.POST("/v1/audio/speech", openAIHandler, ttsHandler.HandleOpenAITTS)
	baseRouter.POST("/audio/speech", openAIHandler, ttsHandler.HandleOpenAITTS)

	// 设置性能监控和健康检查路由
	baseRouter.GET("/metrics", metricsHandler.GetMetrics)
	baseRouter.POST("/metrics/reset", metricsHandler.ResetMetrics)
	baseRouter.GET("/health", metricsHandler.HealthCheck)

	return router, nil
}

// InitializeServices 初始化所有服务
func InitializeServices(cfg *config.Config, logger zerolog.Logger) (tts.Service, error) {
	// 创建Microsoft TTS客户端
	ttsClient := microsoft.NewClient(cfg, logger)

	return ttsClient, nil
}
