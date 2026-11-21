package main

import (
	"log"

	"github.com/jiangbyte/agentdeck/internal/config"
	"github.com/jiangbyte/agentdeck/internal/infra"
	agentcmd "github.com/jiangbyte/agentdeck/internal/modular/agent/command"
	agentcron "github.com/jiangbyte/agentdeck/internal/modular/agent/cron"
	agentdisk "github.com/jiangbyte/agentdeck/internal/modular/agent/disk"
	agentfiles "github.com/jiangbyte/agentdeck/internal/modular/agent/files"
	agentfw "github.com/jiangbyte/agentdeck/internal/modular/agent/firewall"
	agentmarket "github.com/jiangbyte/agentdeck/internal/modular/agent/market"
	agentmon "github.com/jiangbyte/agentdeck/internal/modular/agent/monitor"
	agentproc "github.com/jiangbyte/agentdeck/internal/modular/agent/process"
	agentscript "github.com/jiangbyte/agentdeck/internal/modular/agent/script"
	agentstart "github.com/jiangbyte/agentdeck/internal/modular/agent/startup"
	agenttermcmd "github.com/jiangbyte/agentdeck/internal/modular/agent/termcmd"
	agenttermhost "github.com/jiangbyte/agentdeck/internal/modular/agent/termhost"
	agentterm "github.com/jiangbyte/agentdeck/internal/modular/agent/terminal"
	"github.com/jiangbyte/agentdeck/internal/modular/auth/login"
	"github.com/jiangbyte/agentdeck/internal/modular/auth/session"
	"github.com/jiangbyte/agentdeck/internal/modular/auth/user"
	"github.com/jiangbyte/agentdeck/internal/modular/crypto/key"
	"github.com/jiangbyte/agentdeck/internal/modular/system/host"
	"github.com/jiangbyte/agentdeck/internal/pkg/secret"
	"github.com/jiangbyte/agentdeck/internal/server"
)

func main() {
	// 1. 打开 SQLite（自动建目录与表）
	db, err := infra.OpenDB()
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 2. 按领域模块组装仓储与服务
	loginSvc := login.NewService(user.NewRepo(db), session.NewRepo(db))
	keySvc := key.NewService(key.NewRepo(db))
	hostSvc := host.NewService()
	procSvc := agentproc.NewService()
	cronSvc := agentcron.NewService(agentcron.NewRepo(db))
	startSvc := agentstart.NewService(agentstart.NewRepo(db))
	cmdSvc := agentcmd.NewService()
	marketSvc := agentmarket.NewService(agentmarket.NewRepo(db))
	if err := marketSvc.EnsureSeed(); err != nil {
		log.Printf("应用目录初始化警告: %v", err)
	}
	monSvc := agentmon.NewService()
	fwSvc := agentfw.NewService()
	diskSvc := agentdisk.NewService()
	filesSvc := agentfiles.NewService()
	termHostSvc := agenttermhost.NewService(agenttermhost.NewRepo(db))
	termCmdSvc := agenttermcmd.NewService(agenttermcmd.NewRepo(db))
	scriptSvc := agentscript.NewService(agentscript.NewRepo(db), termHostSvc)
	termSvc := agentterm.NewService(termHostSvc)

	// 3. 确保 SM2 密钥、终端凭证密钥与默认管理员存在
	if err := keySvc.EnsureKey(); err != nil {
		log.Fatalf("初始化国密密钥失败: %v", err)
	}
	if err := secret.EnsureKey(); err != nil {
		log.Fatalf("初始化终端凭证密钥失败: %v", err)
	}
	if err := loginSvc.EnsureAdmin(); err != nil {
		log.Fatalf("初始化管理员失败: %v", err)
	}
	_ = loginSvc.DeleteExpiredSessions()

	// 4. 启动时将定时任务 / 自启条目与系统状态对账
	if err := cronSvc.Reconcile(); err != nil {
		log.Printf("定时任务同步警告: %v", err)
	}
	if err := startSvc.Reconcile(); err != nil {
		log.Printf("开机自启同步警告: %v", err)
	}

	// 5. 启动 HTTP 服务（无外部配置文件）
	engine := server.New(server.Deps{
		Login: loginSvc, Key: keySvc, Host: hostSvc,
		Process: procSvc, Cron: cronSvc, Startup: startSvc, Command: cmdSvc,
		Market: marketSvc, Monitor: monSvc, Firewall: fwSvc, Disk: diskSvc, Files: filesSvc,
		TermHost: termHostSvc, TermCmd: termCmdSvc, Script: scriptSvc, Terminal: termSvc,
	})
	log.Printf("AgentDeck 已启动，监听 %s，数据库 %s", config.ListenAddr, config.DBPath())
	if err := engine.Run(config.ListenAddr); err != nil {
		log.Fatalf("服务退出: %v", err)
	}
}
