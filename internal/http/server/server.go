package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
	"tts/internal/config"
)

// Server 封装HTTP服务器
type Server struct {
	httpServer *http.Server
}

// New 创建新的HTTP服务器
func New(cfg *config.Config, router *gin.Engine) *Server {
	addr := fmt.Sprintf(":%d", cfg.Server.Port)

	readTimeout := time.Duration(cfg.Server.ReadTimeout) * time.Second
	writeTimeout := time.Duration(cfg.Server.WriteTimeout) * time.Second
	if readTimeout <= 0 {
		readTimeout = 60 * time.Second
	}
	if writeTimeout <= 0 {
		writeTimeout = 60 * time.Second
	}

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MiB
	}
	return &Server{
		httpServer: httpServer,
	}
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	fmt.Println("正在关闭HTTP服务器...")
	return s.httpServer.Shutdown(ctx)
}
