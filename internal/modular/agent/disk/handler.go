package disk

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

// Handler 磁盘管理接口。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List 返回磁盘与分区。
func (h *Handler) List(c *gin.Context) {
	response.OK(c, h.svc.List())
}
