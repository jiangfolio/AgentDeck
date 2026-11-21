package market

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jiangbyte/agentdeck/internal/pkg/platform/apps"
)

const maxUploadBytes = 512 << 20 // 512MB

var appIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

// Service 离线绿色包运行框架：一应用一目录，文件自管。
type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) findApp(id string) (*CatalogApp, error) {
	item, err := s.repo.FindCatalog(id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("目录中不存在应用: %s", id)
	}
	return item, nil
}

/**
 * Catalog 分页查询目录，并叠加运行状态与应用目录路径。
 */
func (s *Service) Catalog(q CatalogQuery) (*CatalogResult, error) {
	// 1. 校正分页
	page := q.Page
	if page < 1 {
		page = 1
	}
	size := q.PageSize
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	keyword := strings.TrimSpace(q.Keyword)
	category := strings.TrimSpace(q.Category)

	// 2. 一次取出运行状态
	runtimes, err := s.repo.ListRuntimes()
	if err != nil {
		return nil, err
	}
	rtMap := map[string]Runtime{}
	for _, rt := range runtimes {
		rtMap[rt.AppID] = rt
	}

	// 3. 数据库分页查目录
	rows, total, err := s.repo.ListCatalog(page, size, category, keyword)
	if err != nil {
		return nil, err
	}
	categories, _ := s.repo.DistinctCategories()

	items := make([]App, 0, len(rows))
	for _, row := range rows {
		app := App{
			ID:       row.ID,
			Name:     row.Name,
			Summary:  row.Summary,
			Category: row.Category,
			Bin:      row.Bin,
			Start:    row.Start,
			Env:      row.Env,
			Sort:     row.Sort,
			Runnable: row.Runnable,
			AppDir:   apps.AppDir(row.ID),
		}
		if app.Start == nil {
			app.Start = []string{}
		}
		if rt, ok := rtMap[row.ID]; ok {
			running, pid := apps.StatusByPid(rt.Pid)
			app.Running = running
			app.Pid = pid
			if rt.Pid > 0 && !running {
				_ = s.repo.UpsertPid(row.ID, 0)
			}
		}
		items = append(items, app)
	}
	if categories == nil {
		categories = []string{}
	}
	return &CatalogResult{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   size,
		Categories: categories,
	}, nil
}

/**
 * SaveCatalog 创建或更新目录项，并确保应用目录存在。
 */
func (s *Service) SaveCatalog(req SaveCatalogRequest) (*CatalogApp, error) {
	id := strings.ToLower(strings.TrimSpace(req.ID))
	if !appIDPattern.MatchString(id) {
		return nil, fmt.Errorf("应用 id 须为小写字母开头，仅含小写字母/数字/_/-")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("名称不能为空")
	}
	bin := strings.TrimSpace(req.Bin)
	if bin == "" {
		return nil, fmt.Errorf("二进制相对路径不能为空")
	}
	start := cleanStringList(req.Start)
	if req.Runnable && len(start) == 0 {
		start = []string{"{bin}"}
	}
	if !req.Runnable {
		start = nil
	}

	existing, err := s.repo.FindCatalog(id)
	if err != nil {
		return nil, err
	}
	item := &CatalogApp{
		ID:       id,
		Name:     name,
		Summary:  strings.TrimSpace(req.Summary),
		Category: strings.TrimSpace(req.Category),
		Bin:      bin,
		Start:    start,
		Env:      req.Env,
		Runnable: req.Runnable,
		Sort:     req.Sort,
	}
	if existing != nil {
		if err := s.repo.UpdateCatalog(item); err != nil {
			return nil, err
		}
	} else {
		if err := s.repo.CreateCatalog(item); err != nil {
			return nil, err
		}
	}
	if _, err := apps.EnsureAppDir(id); err != nil {
		return nil, err
	}
	return s.repo.FindCatalog(id)
}

/**
 * RemoveCatalog 删除目录项；若在运行则先停止，并删除应用目录。
 */
func (s *Service) RemoveCatalog(id string) error {
	id = strings.TrimSpace(id)
	item, err := s.repo.FindCatalog(id)
	if err != nil {
		return err
	}
	if item == nil {
		return fmt.Errorf("目录中不存在应用: %s", id)
	}
	rt, _ := s.repo.GetRuntime(id)
	if rt != nil && rt.Pid > 0 {
		_ = apps.Stop(rt.Pid)
	}
	_ = s.repo.DeleteRuntime(id)
	_ = os.RemoveAll(apps.AppDir(id))
	_ = os.RemoveAll(apps.ServiceDir(id))
	return s.repo.DeleteCatalog(id)
}

func cleanStringList(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

/**
 * UploadFiles 将官方文件原样写入应用目录（不解压），返回目录路径。
 */
func (s *Service) UploadFiles(appID string, headers []*multipart.FileHeader) (*UploadResult, error) {
	appID = strings.TrimSpace(appID)
	if _, err := s.findApp(appID); err != nil {
		return nil, err
	}
	if len(headers) == 0 {
		return nil, fmt.Errorf("请选择要上传的文件")
	}
	dir, err := apps.EnsureAppDir(appID)
	if err != nil {
		return nil, err
	}

	saved := make([]string, 0, len(headers))
	for _, fh := range headers {
		name := filepath.Base(fh.Filename)
		if name == "" || name == "." || name == ".." {
			return nil, fmt.Errorf("非法文件名")
		}
		if fh.Size < 0 || fh.Size > maxUploadBytes {
			return nil, fmt.Errorf("文件 %s 超过 512MB 上限", name)
		}
		target, err := apps.SafeJoinUnder(dir, name)
		if err != nil {
			return nil, err
		}
		if err := saveUpload(fh, target); err != nil {
			return nil, err
		}
		saved = append(saved, name)
	}
	return &UploadResult{AppDir: dir, Files: saved}, nil
}

func saveUpload(fh *multipart.FileHeader, target string) error {
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, io.LimitReader(src, maxUploadBytes+1))
	return err
}

/**
 * OpenDir 确保目录存在并返回路径（供跳转文件管理）。
 */
func (s *Service) OpenDir(id string) (string, error) {
	if _, err := s.findApp(id); err != nil {
		return "", err
	}
	return apps.EnsureAppDir(id)
}

/**
 * ClearDir 停止进程并清空应用目录内容（保留空目录）。
 */
func (s *Service) ClearDir(id string) error {
	app, err := s.findApp(id)
	if err != nil {
		return err
	}
	rt, _ := s.repo.GetRuntime(id)
	if rt != nil && rt.Pid > 0 {
		_ = apps.Stop(rt.Pid)
		_ = s.repo.UpsertPid(id, 0)
	}
	dir := apps.AppDir(app.ID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			_, err = apps.EnsureAppDir(id)
			return err
		}
		return err
	}
	for _, e := range entries {
		_ = os.RemoveAll(filepath.Join(dir, e.Name()))
	}
	return nil
}

func (s *Service) Start(id string) error {
	app, err := s.findApp(id)
	if err != nil {
		return err
	}
	rt, err := s.repo.GetRuntime(id)
	if err != nil {
		return err
	}
	if rt != nil {
		if running, _ := apps.StatusByPid(rt.Pid); running {
			return fmt.Errorf("服务已在运行")
		}
	}
	pid, err := apps.Start(id, app.Layout())
	if err != nil {
		return err
	}
	return s.repo.UpsertPid(id, pid)
}

func (s *Service) Stop(id string) error {
	rt, err := s.repo.GetRuntime(id)
	if err != nil {
		return err
	}
	pid := 0
	if rt != nil {
		pid = rt.Pid
	}
	_ = apps.Stop(pid)
	return s.repo.UpsertPid(id, 0)
}

func (s *Service) AppStatus(id string) (*StatusResult, error) {
	if _, err := s.findApp(id); err != nil {
		return nil, err
	}
	rt, err := s.repo.GetRuntime(id)
	if err != nil {
		return nil, err
	}
	pid := 0
	if rt != nil {
		pid = rt.Pid
	}
	running, alive := apps.StatusByPid(pid)
	if pid > 0 && !running {
		_ = s.repo.UpsertPid(id, 0)
	}
	return &StatusResult{
		ID:      id,
		Running: running,
		Pid:     alive,
		AppDir:  apps.AppDir(id),
	}, nil
}
