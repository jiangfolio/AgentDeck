package terminal

import "github.com/gin-gonic/gin"

// Handler 终端接口。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) WS(c *gin.Context) {
	h.svc.HandleWS(c)
}
