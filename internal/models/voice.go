package models

// Voice 表示一个语音合成声音
type Voice struct {
	Name            string   `json:"Name"`            // 语音唯一标识符
	DisplayName     string   `json:"DisplayName"`     // 语音显示名称
	LocalName       string   `json:"LocalName"`       // 本地化名称
	ShortName       string   `json:"ShortName"`       // 简称，例如 zh-CN-XiaoxiaoNeural
	Gender          string   `json:"Gender"`          // 性别: Female, Male
	Locale          string   `json:"Locale"`          // 语言区域, 如 zh-CN
	LocaleName      string   `json:"LocaleName"`      // 语言区域显示名称，如 中文(中国)
	StyleList       []string `json:"StyleList,omitempty"` // 支持的说话风格列表
	SampleRateHertz string   `json:"SampleRateHertz"` // 采样率
}
