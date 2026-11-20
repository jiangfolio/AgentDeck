package startup

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

// Handler 开机自启接口。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Self(c *gin.Context) {
	data, err := h.svc.Self()
	if err != nil {
		response.Fail(c, 500, err.Error())
		return
	}
	response.OK(c, data)
}

func (h *Handler) SelfSet(c *gin.Context) {
	var req SetSelfRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.SetSelf(req.Enabled); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) List(c *gin.Context) {
	var q ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Fail(c, 400, "分页参数无效")
		return
	}
	data, err := h.svc.List(q)
	if err != nil {
		response.Fail(c, 500, "查询失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) Save(c *gin.Context) {
	var req SaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	data, err := h.svc.Save(req)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, data)
}

func (h *Handler) Remove(c *gin.Context) {
	var req IDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Remove(req.ID); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) SetEnabled(c *gin.Context) {
	var req SetEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.SetEnabled(req.ID, req.Enabled); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}
