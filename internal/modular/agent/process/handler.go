package process

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

// Handler 进程管理接口。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List 全量列出进程（支持 keyword/user 筛选，不分页）。
func (h *Handler) List(c *gin.Context) {
	var q ListQuery
	_ = c.ShouldBindQuery(&q)
	data, err := h.svc.List(q)
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.OK(c, data)
}

// Detail 查询进程详情。
func (h *Handler) Detail(c *gin.Context) {
	var q DetailQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, 400, "pid 无效")
		return
	}
	data, err := h.svc.Detail(q.PID)
	if err != nil {
		response.Fail(c, 404, err.Error())
		return
	}
	response.OK(c, data)
}

// Kill 结束进程。
func (h *Handler) Kill(c *gin.Context) {
	var req KillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Kill(req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}
