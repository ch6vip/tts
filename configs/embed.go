package configs

import _ "embed"

// DefaultConfig 保存了嵌入的默认 config.yaml 文件。
//go:embed config.yaml
var DefaultConfig []byte