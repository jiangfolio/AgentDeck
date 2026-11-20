package login

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/config"
	"github.com/jiangbyte/agentdeck/internal/middleware"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

// Handler 登录模块接口。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userVO struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// Login 账号密码登录并写入 Session Cookie。
func (h *Handler) Login(c *gin.Context) {
	// 1. 解析并校验入参
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if req.Username == "" || req.Password == "" {
		response.Fail(c, 400, "用户名或密码不能为空")
		return
	}
	// 2. 校验凭证并创建会话
	result, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		response.Fail(c, 401, err.Error())
		return
	}
	// 3. 写 HttpOnly Cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     config.SessionCookie,
		Value:    result.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(config.SessionTTL),
		MaxAge:   int(config.SessionTTL.Seconds()),
	})
	response.OK(c, userVO{
		ID:          result.User.ID,
		Username:    result.User.Username,
		DisplayName: result.User.DisplayName,
	})
}

// Logout 清除会话。
func (h *Handler) Logout(c *gin.Context) {
	token, _ := c.Cookie(config.SessionCookie)
	_ = h.svc.Logout(token)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     config.SessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	response.OK(c, nil)
}

// Me 返回当前登录用户。
func (h *Handler) Me(c *gin.Context) {
	u := middleware.CurrentUser(c)
	if u == nil {
		token, _ := c.Cookie(config.SessionCookie)
		got, err := h.svc.CurrentUser(token)
		if err != nil {
			response.Fail(c, 500, "查询失败")
			return
		}
		u = got
	}
	if u == nil {
		response.Fail(c, 401, "未登录")
		return
	}
	response.OK(c, userVO{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
	})
}
