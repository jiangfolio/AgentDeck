package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/modular/auth/user"
	"github.com/jiangbyte/agentdeck/internal/config"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

const ctxUser = "current_user"

// SessionUserResolver 由登录模块实现，避免中间件依赖具体 Service 类型造成循环引用。
type SessionUserResolver interface {
	CurrentUser(token string) (*user.User, error)
}

// SessionAuth 校验 Cookie 会话，未登录则拒绝。
func SessionAuth(resolver SessionUserResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie(config.SessionCookie)
		u, err := resolver.CurrentUser(token)
		if err != nil {
			response.AbortFail(c, http.StatusInternalServerError, 500, "会话校验失败")
			return
		}
		if u == nil {
			response.AbortFail(c, http.StatusUnauthorized, 401, "未登录")
			return
		}
		c.Set(ctxUser, u)
		c.Next()
	}
}

// CurrentUser 从上下文取当前用户。
func CurrentUser(c *gin.Context) *user.User {
	v, ok := c.Get(ctxUser)
	if !ok {
		return nil
	}
	u, _ := v.(*user.User)
	return u
}
