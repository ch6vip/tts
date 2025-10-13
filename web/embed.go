package web

import "embed"

// TemplatesFS 包含所有 HTML 模板
//go:embed all:templates
var TemplatesFS embed.FS

// StaticFS 包含所有静态文件
//go:embed all:static
var StaticFS embed.FS