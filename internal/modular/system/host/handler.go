package host

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

// Handler 系统主机信息接口。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Overview 返回跨平台主机与资源概览。
func (h *Handler) Overview(c *gin.Context) {
	data, err := h.svc.Overview()
	if err != nil {
		response.Fail(c, 500, "获取系统信息失败")
		return
	}
	response.OK(c, data)
}
