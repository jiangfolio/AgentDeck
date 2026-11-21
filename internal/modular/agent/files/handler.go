package files

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

const maxUploadBody = maxUploadBytes + 1<<20

// Handler 文件管理接口。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Roots(c *gin.Context) {
	response.OK(c, h.svc.Roots())
}

func (h *Handler) List(c *gin.Context) {
	var q ListQuery
	_ = c.ShouldBindQuery(&q)
	res, err := h.svc.List(q)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, res)
}

func (h *Handler) Read(c *gin.Context) {
	path := strings.TrimSpace(c.Query("path"))
	res, err := h.svc.Read(path)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, res)
}

func (h *Handler) Write(c *gin.Context) {
	var req WriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Write(req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Mkdir(c *gin.Context) {
	var req NameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Mkdir(req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Create(c *gin.Context) {
	var req NameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Create(req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Rename(c *gin.Context) {
	var req RenameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Rename(req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Remove(c *gin.Context) {
	var req PathsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Remove(req.Paths); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) TrashList(c *gin.Context) {
	var q TrashQuery
	_ = c.ShouldBindQuery(&q)
	res, err := h.svc.TrashList(q)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, res)
}

func (h *Handler) TrashRestore(c *gin.Context) {
	var req TrashIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.TrashRestore(req.IDs); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) TrashPurge(c *gin.Context) {
	var req TrashIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.TrashPurge(req.IDs); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) TrashEmpty(c *gin.Context) {
	if err := h.svc.TrashEmpty(); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Copy(c *gin.Context) {
	var req CopyMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Copy(req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Move(c *gin.Context) {
	var req CopyMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 400, "请求参数无效")
		return
	}
	if err := h.svc.Move(req); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBody)
	dir := strings.TrimSpace(c.PostForm("path"))
	fh, err := c.FormFile("file")
	if err != nil {
		if tooLarge(err) {
			response.Fail(c, 400, "上传文件不能超过 64MB")
			return
		}
		response.Fail(c, 400, "请上传文件")
		return
	}
	if fh.Size < 0 || fh.Size > maxUploadBytes {
		response.Fail(c, 400, "上传文件不能超过 64MB")
		return
	}
	src, err := fh.Open()
	if err != nil {
		response.Fail(c, 400, "读取上传文件失败")
		return
	}
	defer src.Close()
	name := filepath.Base(fh.Filename)
	if err := h.svc.Upload(dir, name, src, fh.Size); err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *Handler) Download(c *gin.Context) {
	path := strings.TrimSpace(c.Query("path"))
	f, st, err := h.svc.OpenDownload(path)
	if err != nil {
		response.Fail(c, 400, err.Error())
		return
	}
	defer f.Close()
	name := filepath.Base(path)
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = filepath.Base(st.Name())
	}
	c.Header("Content-Disposition", "attachment; filename=\""+strings.ReplaceAll(name, "\"", "")+"\"")
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.FormatInt(st.Size(), 10))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, f)
}

func tooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr) || strings.Contains(strings.ToLower(err.Error()), "too large")
}

func urlPathEscape(name string) string {
	replacer := strings.NewReplacer(
		" ", "%20",
		"\"", "%22",
		"'", "%27",
		";", "%3B",
	)
	return replacer.Replace(name)
}
