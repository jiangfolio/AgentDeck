package webembed

import "embed"

// Dist 嵌入前端构建产物；发布前由 Makefile 写入 dist。
//
//go:embed all:dist
var Dist embed.FS
