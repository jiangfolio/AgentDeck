package monitor

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

// Handler 监控接口。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Snapshot 返回当前一拍监控数据。
func (h *Handler) Snapshot(c *gin.Context) {
	data, err := h.svc.Snapshot()
	if err != nil {
		response.Fail(c, 500, "采集监控数据失败")
		return
	}
	response.OK(c, data)
}
