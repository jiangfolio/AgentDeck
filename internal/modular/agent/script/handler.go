package script

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

// Handler 脚本库接口。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListGroups(c *gin.Context) {
	data, err := h.svc.ListGroups()
	if err != nil {
		response.Fail(c, 500, "查询分组失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) SaveGroup(c *gin.Context) {
	var req SaveGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	data, err := h.svc.SaveGroup(req)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, data)
}

func (h *Handler) RemoveGroup(c *gin.Context) {
	var req IDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.RemoveGroup(req.ID); err != nil {
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
	var req SaveScriptRequest
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
	var req IDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Remove(req.IDs); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

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
