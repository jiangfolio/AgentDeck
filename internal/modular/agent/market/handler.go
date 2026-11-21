package market

import (
	"errors"
	"net/http"
	"strings"

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
	data, err := h.svc.Catalog(q)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, data)
}

/**
 * SaveCatalog 创建或更新目录项。
 */
func (h *Handler) SaveCatalog(c *gin.Context) {
	var req SaveCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	item, err := h.svc.SaveCatalog(req)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, item)
}

/**
 * RemoveCatalog 删除目录项及应用目录。
 */
func (h *Handler) RemoveCatalog(c *gin.Context) {
	var req ActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.RemoveCatalog(req.ID); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

/**
 * Upload 上传官方文件到应用目录（不解压）。
 */
func (h *Handler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes*4+1<<20)
	id := strings.TrimSpace(c.PostForm("id"))
	form, err := c.MultipartForm()
	if err != nil {
		if tooLarge(err) {
			response.Fail(c, 400, "上传体积过大")
			return
		}
		response.Fail(c, 400, "请上传文件")
		return
	}
	files := form.File["file"]
	if len(files) == 0 {
		response.Fail(c, 400, "请上传文件")
		return
	}
	result, err := h.svc.UploadFiles(id, files)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, result)
}

/**
 * OpenDir 返回应用目录路径。
 */
func (h *Handler) OpenDir(c *gin.Context) {
	var req ActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	dir, err := h.svc.OpenDir(req.ID)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, gin.H{"app_dir": dir})
}

/**
 * ClearDir 清空应用目录内容。
 */
func (h *Handler) ClearDir(c *gin.Context) {
	var req ActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.ClearDir(req.ID); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Start(c *gin.Context) {
	var req ActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Start(req.ID); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Stop(c *gin.Context) {
	var req ActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Stop(req.ID); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Status(c *gin.Context) {
	id := strings.TrimSpace(c.Query("id"))
	if id == "" {
		response.Fail(c, 400, "缺少 id")
		return
	}
	st, err := h.svc.AppStatus(id)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, st)
}

func tooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr) || strings.Contains(strings.ToLower(err.Error()), "too large")
}
