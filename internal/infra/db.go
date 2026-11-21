package infra

import (
	"fmt"
	"os"

	"github.com/glebarez/sqlite"
	"github.com/jiangbyte/agentdeck/internal/config"
	"github.com/jiangbyte/agentdeck/internal/modular/agent/cron"
	"github.com/jiangbyte/agentdeck/internal/modular/agent/market"
	"github.com/jiangbyte/agentdeck/internal/modular/agent/script"
	"github.com/jiangbyte/agentdeck/internal/modular/agent/startup"
	"github.com/jiangbyte/agentdeck/internal/modular/agent/termcmd"
	"github.com/jiangbyte/agentdeck/internal/modular/agent/termhost"
	"github.com/jiangbyte/agentdeck/internal/modular/auth/session"
	"github.com/jiangbyte/agentdeck/internal/modular/auth/user"
	"github.com/jiangbyte/agentdeck/internal/modular/crypto/key"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OpenDB 创建 data 目录并打开 SQLite（纯 Go 驱动，便于跨平台）。
func OpenDB() (*gorm.DB, error) {
	// 1. 确保数据目录存在
	if err := os.MkdirAll(config.DataDir(), 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	// 2. 打开数据库（modernc 纯 Go sqlite，无 CGO）
	db, err := gorm.Open(sqlite.Open(config.DBPath()), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Error),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	// 3. 自动建表（各领域模块模型）
	if err := db.AutoMigrate(
		&user.User{}, &session.Session{}, &key.Key{},
		&cron.Job{}, &startup.Entry{},
		&termhost.HostGroup{}, &termhost.Host{},
		&termcmd.QuickCmd{},
		&script.Group{}, &script.Script{},
		&market.CatalogApp{}, &market.Runtime{},
	); err != nil {
		return nil, fmt.Errorf("迁移表结构失败: %w", err)
	}
	return db, nil
}
