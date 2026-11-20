package market

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

// Handler 应用市场接口。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Catalog 分页返回应用目录。
func (h *Handler) Catalog(c *gin.Context) {
	var q CatalogQuery
	_ = c.ShouldBindQuery(&q)
	response.OK(c, h.svc.Catalog(q))
}
