package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"tts/internal/config"
	"tts/internal/http/server"
)

// initLog 初始化日志记录器
func initLog(logConfig *config.LogConfig) {
	// 设置日志格式
	if logConfig.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	// 设置日志级别
	level, err := logrus.ParseLevel(logConfig.Level)
	if err != nil {
		logrus.WithError(err).Warnf("无效的日志级别 '%s'，回退到 'info'", logConfig.Level)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// 设置日志输出
	logrus.SetOutput(os.Stdout)
}

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	flag.Parse()

	// 如果没有指定配置文件，尝试默认位置
	if *configPath == "" {
		// 尝试多个位置查找配置文件
		possiblePaths := []string{
			"./configs/config.yaml",
			"../configs/config.yaml",
			"/etc/tts/config.yaml",
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				*configPath = path
				break
			}
		}

		// 如果还是没找到，使用默认位置
		if *configPath == "" {
			*configPath = "./configs/config.yaml"
		}
	}

	// 确保配置文件路径是绝对路径
	absConfigPath, err := filepath.Abs(*configPath)
	if err != nil {
		logrus.Fatalf("无法获取配置文件的绝对路径: %v", err)
	}

	// 打印使用的配置文件路径
	// 加载配置
	cfg, err := config.Load(absConfigPath)
	if err != nil {
		logrus.Fatalf("无法加载配置: %v", err)
	}

	// 初始化日志
	initLog(&cfg.Log)

	logrus.Infof("使用配置文件: %s", absConfigPath)

	// 创建并启动应用
	app, err := server.NewApp(cfg)
	if err != nil {
		logrus.Fatalf("初始化应用失败: %v", err)
	}

	// 启动应用并处理错误
	if err := app.Start(); err != nil {
		logrus.Fatalf("应用运行出错: %v", err)
	}
}
