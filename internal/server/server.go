package server

import (
	"io/fs"
	"net/http"
	"strings"

	agentcmd "github.com/jiangbyte/agentdeck/internal/modular/agent/command"
	agentcron "github.com/jiangbyte/agentdeck/internal/modular/agent/cron"
	agentproc "github.com/jiangbyte/agentdeck/internal/modular/agent/process"
	agentstart "github.com/jiangbyte/agentdeck/internal/modular/agent/startup"
	"github.com/jiangbyte/agentdeck/internal/modular/auth/login"
	"github.com/jiangbyte/agentdeck/internal/modular/crypto/key"
	"github.com/jiangbyte/agentdeck/internal/middleware"
	"github.com/jiangbyte/agentdeck/internal/modular/system/host"
	"github.com/jiangbyte/agentdeck/internal/webembed"
	"github.com/gin-gonic/gin"
)

// Deps 组装路由所需依赖。
type Deps struct {
	Login   *login.Service
	Key     *key.Service
	Host    *host.Service
	Process *agentproc.Service
	Cron    *agentcron.Service
	Startup *agentstart.Service
	Command *agentcmd.Service
}

// New 创建 Gin 引擎并注册 API 与 SPA。
func New(deps Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	loginH := login.NewHandler(deps.Login)
	keyH := key.NewHandler(deps.Key)
	hostH := host.NewHandler(deps.Host)
	procH := agentproc.NewHandler(deps.Process)
	cronH := agentcron.NewHandler(deps.Cron)
	startH := agentstart.NewHandler(deps.Startup)
	cmdH := agentcmd.NewHandler(deps.Command)

	api := r.Group("/api")
	// 仅登录接口走国密加解密；其余 API 明文 JSON
	api.Use(middleware.LoginEncrypt(deps.Key))
	{
		api.GET("/crypto/public-key", keyH.PublicKey)
		api.POST("/auth/login", loginH.Login)
		api.POST("/auth/logout", loginH.Logout)

		authRequired := api.Group("")
		authRequired.Use(middleware.SessionAuth(deps.Login))
		{
			authRequired.GET("/auth/me", loginH.Me)
			authRequired.GET("/system/host", hostH.Overview)

			authRequired.GET("/agent/process/list", procH.List)
			authRequired.GET("/agent/process/detail", procH.Detail)
			authRequired.POST("/agent/process/kill", procH.Kill)

			authRequired.GET("/agent/cron/list", cronH.List)
			authRequired.POST("/agent/cron/save", cronH.Save)
			authRequired.POST("/agent/cron/remove", cronH.Remove)
			authRequired.POST("/agent/cron/set_enabled", cronH.SetEnabled)
			authRequired.GET("/agent/cron/runs", cronH.Runs)

			authRequired.GET("/agent/startup/self", startH.Self)
			authRequired.POST("/agent/startup/self_set", startH.SelfSet)
			authRequired.GET("/agent/startup/list", startH.List)
			authRequired.POST("/agent/startup/save", startH.Save)
			authRequired.POST("/agent/startup/remove", startH.Remove)
			authRequired.POST("/agent/startup/set_enabled", startH.SetEnabled)

			authRequired.POST("/agent/command/run", cmdH.Run)
		}
	}

	mountSPA(r)
	return r
}

// mountSPA 托管嵌入的前端静态资源，未知路径回退到 index.html。
func mountSPA(r *gin.Engine) {
	sub, err := fs.Sub(webembed.Dist, "dist")
	if err != nil {
		return
	}
	fileServer := http.FileServer(http.FS(sub))
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") {
			c.Status(http.StatusNotFound)
			return
		}
		f, err := sub.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
