package config

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/spf13/viper"
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
}

var (
	config Config
	once   sync.Once
)

// Load 从指定路径加载配置文件
func Load(configPath string) (*Config, error) {
	var err error
	once.Do(func() {
		v := viper.New()

		// 配置 Viper
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv() // 自动绑定环境变量

		// 从配置文件加载
		if configPath != "" {
			v.SetConfigFile(configPath)
			if err = v.ReadInConfig(); err != nil {
				err = fmt.Errorf("加载配置文件失败: %w", err)
				return
			}
		}

		// 将配置绑定到结构体
		if err = v.Unmarshal(&config); err != nil {
			err = fmt.Errorf("解析配置失败: %w", err)
			return
		}

	})

	if err != nil {
		return nil, err
	}

	return &config, nil
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
