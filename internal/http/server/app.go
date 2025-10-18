package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"tts/internal/config"
	"tts/internal/http/routes"
	"tts/internal/tts"
)

// App 表示整个TTS应用程序
type App struct {
	server     *Server
	cfg        *config.Config
	ttsService tts.Service
}

// NewApp 创建一个新的应用程序实例
func NewApp(cfg *config.Config) (*App, error) {
	// 初始化服务
	ttsService, err := routes.InitializeServices(cfg)
	if err != nil {
		return nil, fmt.Errorf("初始化服务失败: %w", err)
	}

	// 如果启用了缓存，则包装原始服务
	if cfg.Cache.Enabled {
		logrus.Info("启用TTS缓存")
		ttsService = tts.NewCachingService(
			ttsService,
			time.Duration(cfg.Cache.ExpirationMinutes)*time.Minute,
			time.Duration(cfg.Cache.CleanupIntervalMinutes)*time.Minute,
		)
	}

	// 设置Gin路由
	router, err := routes.SetupRoutes(cfg, ttsService)
	if err != nil {
		return nil, fmt.Errorf("设置路由失败: %w", err)
	}

	// 创建HTTP服务器
	server := New(cfg, router)

	return &App{
		server:     server,
		cfg:        cfg,
		ttsService: ttsService,
	}, nil
}

// Start 启动应用程序
func (a *App) Start() error {
	// 创建一个错误通道
	errChan := make(chan error, 1)

	// 创建一个退出信号通道
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 在一个goroutine中启动服务器
	go func() {
		logrus.Infof("启动TTS服务，监听端口 %d...", a.cfg.Server.Port)
		errChan <- a.server.Start()
	}()

	// 等待退出信号或错误
	select {
	case err := <-errChan:
		return err
	case <-quit:
		// 创建一个超时上下文用于优雅关闭
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 尝试优雅关闭服务器
		if err := a.server.Shutdown(ctx); err != nil {
			logrus.Errorf("服务器关闭出错: %v", err)
		}

		// 关闭 TTS 服务（例如，关闭 worker pool）
		if closer, ok := a.ttsService.(interface{ Close() }); ok {
			logrus.Info("正在关闭 TTS 服务...")
			closer.Close()
		}

		logrus.Info("服务器已优雅关闭")
		return nil
	}
}
