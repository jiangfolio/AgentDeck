package firewall

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	plat "github.com/jiangbyte/agentdeck/internal/pkg/platform/firewall"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

// Handler 防火墙接口。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Status(c *gin.Context) {
	response.OK(c, h.svc.Status())
}

func (h *Handler) Rules(c *gin.Context) {
	var q RulesQuery
	_ = c.ShouldBindQuery(&q)
	response.OK(c, h.svc.Rules(q))
}

func (h *Handler) SetEnabled(c *gin.Context) {
	var req EnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.SetEnabled(req.Enabled); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Save(c *gin.Context) {
	var req SaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Save(req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
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

func (h *Handler) Reset(c *gin.Context) {
	if err := h.svc.Reset(); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Export(c *gin.Context) {
	rules, err := h.svc.Export()
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	if rules == nil {
		rules = []plat.Rule{}
	}
	response.OK(c, rules)
}

func (h *Handler) InstallOnline(c *gin.Context) {
	if err := h.svc.InstallOnline(); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) InstallOffline(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFirewallPkg+1<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		if tooLarge(err) {
			response.Fail(c, 400, "安装包不能超过 64MB")
			return
		}
		response.Fail(c, 400, "请上传安装包")
		return
	}
	if fh.Size <= 0 || fh.Size > maxFirewallPkg {
		response.Fail(c, 400, "安装包大小无效")
		return
	}
	src, err := fh.Open()
	if err != nil {
		response.Fail(c, 400, "读取安装包失败")
		return
	}
	defer src.Close()
	if err := h.svc.InstallOffline(fh.Filename, src); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func tooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr) || strings.Contains(strings.ToLower(err.Error()), "too large")
}

func (h *Handler) Import(c *gin.Context) {
	var req ImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Import(req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}
