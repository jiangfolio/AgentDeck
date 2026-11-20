package command

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

// Handler 命令试运行接口。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Run 试运行命令并返回输出。
func (h *Handler) Run(c *gin.Context) {
	var req RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	data, err := h.svc.Run(req)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, data)
}
