package migrations

import "embed"

// Files 打包所有 SQL 迁移，供应用启动时执行。
//
//go:embed sql/*.sql
var Files embed.FS
