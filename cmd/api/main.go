package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"tts/internal/config"
	"tts/internal/http/middleware"
	"tts/internal/http/server"
)


// initLog 初始化日志记录器
func initLog(logConfig *config.LogConfig) {
	// 初始化 logrus（保持向后兼容）
	if logConfig.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	// 设置 logrus 日志级别
	level, err := logrus.ParseLevel(logConfig.Level)
	if err != nil {
		logrus.WithError(err).Warnf("无效的日志级别 '%s'，回退到 'info'", logConfig.Level)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// 设置 logrus 日志输出
	logrus.SetOutput(os.Stdout)
	
	// 初始化 zerolog（高性能日志）
	middleware.InitZerologWithConfig(logConfig)
}

// findProjectRoot 向上遍历目录以查找 go.mod 文件，从而确定项目根目录。
func findProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(cwd, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return cwd, nil
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			// 到达文件系统根目录
			return "", fmt.Errorf("在任何父目录中都未找到 go.mod")
		}
		cwd = parent
	}
}

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	flag.Parse()

	// 如果没有通过 -config 参数指定配置文件，则自动查找
	if *configPath == "" {
		var foundPath string

		// 1. 尝试从项目根目录查找
		if root, err := findProjectRoot(); err == nil {
			path := filepath.Join(root, "configs", "config.yaml")
			if _, err := os.Stat(path); err == nil {
				foundPath = path
				logrus.Debugf("在项目根目录找到配置文件: %s", path)
			}
		}

		// 2. 如果在项目根目录找不到，则检查系统范围的路径
		if foundPath == "" {
			path := "/etc/tts/config.yaml"
			if _, err := os.Stat(path); err == nil {
				foundPath = path
				logrus.Debugf("在系统路径找到配置文件: %s", path)
			}
		}

		// 3. 如果找不到外部配置文件，将使用嵌入的默认配置
		if foundPath == "" {
			logrus.Info("未找到外部配置文件，将使用嵌入的默认配置")
			*configPath = ""
		} else {
			*configPath = foundPath
		}
	}

	var absConfigPath string
	if *configPath != "" {
		// 确保配置文件路径是绝对路径
		var err error
		absConfigPath, err = filepath.Abs(*configPath)
		if err != nil {
			logrus.Fatalf("无法获取配置文件的绝对路径: %v", err)
		}
	}

	// 加载配置（如果找不到外部配置文件，将自动回退到嵌入的默认配置）
	cfg, err := config.Load(absConfigPath)
	if err != nil {
		logrus.Fatalf("无法加载配置: %v", err)
	}

	// 初始化日志
	initLog(&cfg.Log)

	if absConfigPath != "" {
		logrus.Infof("使用配置文件: %s", absConfigPath)
	} else {
		logrus.Info("使用嵌入的默认配置")
	}

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
