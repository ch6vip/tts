package config

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"tts/configs"
)

// Config 包含应用程序的所有配置
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	TTS    TTSConfig    `mapstructure:"tts"`
	OpenAI OpenAIConfig `mapstructure:"openai"`
	SSML   SSMLConfig   `mapstructure:"ssml"`
	Log    LogConfig    `mapstructure:"log"`
	Cache  CacheConfig  `mapstructure:"cache"`
}

// LogConfig 包含日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// CacheConfig 包含缓存配置
type CacheConfig struct {
	Enabled                bool `mapstructure:"enabled"`
	ExpirationMinutes      int  `mapstructure:"expiration_minutes"`
	CleanupIntervalMinutes int  `mapstructure:"cleanup_interval_minutes"`
}

// OpenAIConfig 包含OpenAI API配置
type OpenAIConfig struct {
	ApiKey string `mapstructure:"api_key"`
}

// ServerConfig 包含HTTP服务器配置
type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	BasePath     string `mapstructure:"base_path"`
}

// TTSConfig 包含Microsoft TTS API配置
type TTSConfig struct {
	ApiKey            string            `mapstructure:"api_key"`
	Region            string            `mapstructure:"region"`
	DefaultVoice      string            `mapstructure:"default_voice"`
	DefaultRate       string            `mapstructure:"default_rate"`
	DefaultPitch      string            `mapstructure:"default_pitch"`
	DefaultFormat     string            `mapstructure:"default_format"`
	MaxTextLength     int               `mapstructure:"max_text_length"`
	RequestTimeout    int               `mapstructure:"request_timeout"`
	MaxConcurrent     int               `mapstructure:"max_concurrent"`
	SegmentThreshold  int               `mapstructure:"segment_threshold"`
	MinSentenceLength int               `mapstructure:"min_sentence_length"`
	MaxSentenceLength int               `mapstructure:"max_sentence_length"`
	VoiceMapping      map[string]string `mapstructure:"voice_mapping"`
	
	// 长文本处理配置
	LongText LongTextConfig `mapstructure:"long_text"`
}

// LongTextConfig 长文本 TTS 处理配置
type LongTextConfig struct {
	Enabled            bool   `mapstructure:"enabled"`              // 是否启用长文本优化处理
	MaxSegmentLength   int    `mapstructure:"max_segment_length"`   // 每个分段的最大字符数（默认 500）
	WorkerCount        int    `mapstructure:"worker_count"`         // 并发 worker 数量（默认 5）
	MinTextForSplit    int    `mapstructure:"min_text_for_split"`   // 触发分段的最小文本长度（默认 1000）
	FFmpegPath         string `mapstructure:"ffmpeg_path"`          // FFmpeg 可执行文件路径（留空使用系统 PATH）
	UseSmartSegment    bool   `mapstructure:"use_smart_segment"`    // 是否使用智能分段（基于句子边界）
	UseFFmpegMerge     bool   `mapstructure:"use_ffmpeg_merge"`     // 是否使用 FFmpeg 合并音频（推荐）
}

var (
	config     Config
	configOnce sync.Once
	loadErr    error
)

// Load 从指定路径加载配置文件，如果找不到则使用嵌入的默认配置
func Load(configPath string) (*Config, error) {
	configOnce.Do(func() {
		v := viper.New()

		// 配置 Viper
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv() // 自动绑定环境变量

		// 尝试从配置文件加载
		configLoaded := false
		if configPath != "" {
			v.SetConfigFile(configPath)
			if err := v.ReadInConfig(); err != nil {
				// 配置文件加载失败，尝试使用嵌入的默认配置
				fmt.Printf("警告: 无法加载配置文件 %s: %v，将使用嵌入的默认配置\n", configPath, err)
			} else {
				configLoaded = true
			}
		}

		// 如果没有从文件加载配置，则使用嵌入的默认配置
		if !configLoaded {
			if err := v.ReadConfig(bytes.NewReader(configs.DefaultConfig)); err != nil {
				loadErr = fmt.Errorf("读取嵌入的默认配置失败: %w", err)
				return
			}
			fmt.Println("使用嵌入的默认配置")
		}

		// 将配置绑定到结构体
		if loadErr = v.Unmarshal(&config); loadErr != nil {
			loadErr = fmt.Errorf("解析配置失败: %w", loadErr)
			return
		}

		// 设置默认值
		setDefaults(&config)

		// 验证配置
		if loadErr = validate(&config); loadErr != nil {
			loadErr = fmt.Errorf("配置验证失败: %w", loadErr)
			return
		}
	})

	if loadErr != nil {
		return nil, loadErr
	}

	return &config, nil
}

// setDefaults 设置配置默认值
func setDefaults(cfg *Config) {
	// Server 默认值
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 60
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 60
	}

	// TTS 默认值
	if cfg.TTS.DefaultVoice == "" {
		cfg.TTS.DefaultVoice = "zh-CN-XiaoxiaoNeural"
	}
	if cfg.TTS.DefaultRate == "" {
		cfg.TTS.DefaultRate = "0"
	}
	if cfg.TTS.DefaultPitch == "" {
		cfg.TTS.DefaultPitch = "0"
	}
	if cfg.TTS.DefaultFormat == "" {
		cfg.TTS.DefaultFormat = "audio-24khz-48kbitrate-mono-mp3"
	}
	if cfg.TTS.MaxTextLength == 0 {
		cfg.TTS.MaxTextLength = 65535
	}
	if cfg.TTS.RequestTimeout == 0 {
		cfg.TTS.RequestTimeout = 30
	}
	if cfg.TTS.MaxConcurrent == 0 {
		cfg.TTS.MaxConcurrent = 20
	}

	// 长文本处理默认值
	if cfg.TTS.LongText.MaxSegmentLength == 0 {
		cfg.TTS.LongText.MaxSegmentLength = 500
	}
	if cfg.TTS.LongText.WorkerCount == 0 {
		cfg.TTS.LongText.WorkerCount = 5
	}
	if cfg.TTS.LongText.MinTextForSplit == 0 {
		cfg.TTS.LongText.MinTextForSplit = 1000
	}

	// 日志默认值
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = "text"
	}

	// 缓存默认值
	if cfg.Cache.ExpirationMinutes == 0 {
		cfg.Cache.ExpirationMinutes = 1440
	}
	if cfg.Cache.CleanupIntervalMinutes == 0 {
		cfg.Cache.CleanupIntervalMinutes = 1440
	}
}

// validate 验证配置
func validate(cfg *Config) error {
	// Server 验证
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("无效的服务器端口: %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout < 0 || cfg.Server.WriteTimeout < 0 {
		return fmt.Errorf("超时配置不能为负数")
	}

	// TTS 验证
	if cfg.TTS.MaxTextLength < 1 {
		return fmt.Errorf("max_text_length 必须大于 0")
	}
	if cfg.TTS.RequestTimeout < 1 {
		return fmt.Errorf("request_timeout 必须大于 0")
	}
	if cfg.TTS.MaxConcurrent < 1 {
		return fmt.Errorf("max_concurrent 必须大于 0")
	}
	if cfg.TTS.MaxConcurrent > 100 {
		return fmt.Errorf("max_concurrent 不能超过 100")
	}

	// 长文本处理验证
	if cfg.TTS.LongText.Enabled {
		if cfg.TTS.LongText.MaxSegmentLength < 100 {
			return fmt.Errorf("max_segment_length 不能小于 100")
		}
		if cfg.TTS.LongText.WorkerCount < 1 {
			return fmt.Errorf("worker_count 必须大于 0")
		}
		if cfg.TTS.LongText.WorkerCount > 50 {
			return fmt.Errorf("worker_count 不能超过 50")
		}
		if cfg.TTS.LongText.MinTextForSplit < cfg.TTS.LongText.MaxSegmentLength {
			return fmt.Errorf("min_text_for_split 应大于等于 max_segment_length")
		}
	}

	// 日志级别验证
	validLogLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	levelValid := false
	for _, level := range validLogLevels {
		if cfg.Log.Level == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		return fmt.Errorf("无效的日志级别: %s", cfg.Log.Level)
	}

	// 日志格式验证
	if cfg.Log.Format != "text" && cfg.Log.Format != "json" {
		return fmt.Errorf("无效的日志格式: %s (支持: text, json)", cfg.Log.Format)
	}

	// 缓存验证
	if cfg.Cache.Enabled {
		if cfg.Cache.ExpirationMinutes < 1 {
			return fmt.Errorf("expiration_minutes 必须大于 0")
		}
		if cfg.Cache.CleanupIntervalMinutes < 1 {
			return fmt.Errorf("cleanup_interval_minutes 必须大于 0")
		}
	}

	return nil
}

// Get 返回已加载的配置
func Get() *Config {
	return &config
}

// SSMLConfig 存储SSML标签配置
type SSMLConfig struct {
	// 此结构体现在为空，因为新的处理器会处理所有标签。
	// 保留它是为了与配置结构兼容。
}

// SSMLProcessor 处理SSML内容
type SSMLProcessor struct {
	config *SSMLConfig
}

// NewSSMLProcessor 从配置对象创建SSMLProcessor
func NewSSMLProcessor(config *SSMLConfig) (*SSMLProcessor, error) {
	processor := &SSMLProcessor{
		config: config,
	}
	return processor, nil
}

// EscapeSSML 使用XML解析器安全地转义SSML内容中的文本节点
func (p *SSMLProcessor) EscapeSSML(ssml string) string {
	if ssml == "" {
		return ""
	}
	// 为了处理可能没有根元素的SSML片段，我们将其包装在一个临时的根元素中
	wrappedSSML := "<speak>" + ssml + "</speak>"
	decoder := xml.NewDecoder(strings.NewReader(wrappedSSML))
	decoder.Strict = false // 容忍非标准的XML

	var builder strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			// 如果解析出错，返回原始字符串作为后备
			return ssml
		}

		switch t := token.(type) {
		case xml.StartElement:
			builder.WriteString("<" + t.Name.Local)
			for _, attr := range t.Attr {
				builder.WriteString(fmt.Sprintf(` %s="%s"`, attr.Name.Local, attr.Value))
			}
			builder.WriteString(">")
		case xml.EndElement:
			builder.WriteString("</" + t.Name.Local + ">")
		case xml.CharData:
			// 这是关键：只对文本节点进行转义
			var escapedText bytes.Buffer
			if err := xml.EscapeText(&escapedText, t); err != nil {
				builder.Write(t) // 出错时回退到原始文本
			} else {
				builder.Write(escapedText.Bytes())
			}
		case xml.Comment:
			builder.WriteString("<!--")
			builder.Write(t)
			builder.WriteString("-->")
		case xml.ProcInst:
			builder.WriteString("<?")
			builder.WriteString(t.Target)
			builder.WriteString(" ")
			builder.Write(t.Inst)
			builder.WriteString("?>")
		case xml.Directive:
			builder.WriteString("<!")
			builder.Write(t)
			builder.WriteString(">")
		}
	}

	// 移除我们添加的临时包装
	result := builder.String()
	result = strings.TrimPrefix(result, "<speak>")
	result = strings.TrimSuffix(result, "</speak>")

	return result
}
