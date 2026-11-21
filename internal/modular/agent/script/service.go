package script

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jiangbyte/agentdeck/internal/modular/agent/termhost"
	platterm "github.com/jiangbyte/agentdeck/internal/pkg/platform/terminal"
	"gorm.io/gorm"
)

const runTimeout = 60 * time.Second
const outputLimit = 64000

// Service 脚本库业务。
type Service struct {
	repo  *Repo
	hosts *termhost.Service
}

func NewService(repo *Repo, hosts *termhost.Service) *Service {
	return &Service{repo: repo, hosts: hosts}
}

func (s *Service) ListGroups() ([]Group, error) {
	return s.repo.ListGroups()
}

/**
 * SaveGroup 创建或更新脚本分组。
 */
func (s *Service) SaveGroup(req SaveGroupRequest) (*Group, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("分组名称不能为空")
	}
	if strings.TrimSpace(req.ID) == "" {
		g := &Group{ID: uuid.NewString(), Name: name, Sort: req.Sort, CreatedAt: time.Now()}
		if err := s.repo.CreateGroup(g); err != nil {
			return nil, fmt.Errorf("创建分组失败")
		}
		return g, nil
	}
	g, err := s.repo.FindGroup(req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分组不存在")
		}
		return nil, err
	}
	g.Name = name
	g.Sort = req.Sort
	if err := s.repo.UpdateGroup(g); err != nil {
		return nil, fmt.Errorf("更新分组失败")
	}
	return g, nil
}

/**
 * RemoveGroup 仅允许删除空分组。
 */
func (s *Service) RemoveGroup(id string) error {
	n, err := s.repo.CountByGroup(id)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("分组下仍有脚本，无法删除")
	}
	return s.repo.DeleteGroup(id)
}

/**
 * List 分页列出脚本。
 */
func (s *Service) List(q ListQuery) (*ListResult, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	items, total, err := s.repo.List(q.Page, q.PageSize, q.Keyword, q.GroupID)
	if err != nil {
		return nil, err
	}
	gids := make([]string, 0, len(items))
	for _, it := range items {
		gids = append(gids, it.GroupID)
	}
	names, err := s.repo.MapGroupNames(gids)
	if err != nil {
		return nil, err
	}
	views := make([]ScriptView, 0, len(items))
	for _, it := range items {
		views = append(views, ScriptView{Script: it, GroupName: names[it.GroupID]})
	}
	return &ListResult{Items: views, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

/**
 * Save 创建或更新脚本。
 */
func (s *Service) Save(req SaveScriptRequest) (*ScriptView, error) {
	name := strings.TrimSpace(req.Name)
	content := strings.TrimSpace(req.Content)
	if name == "" || content == "" {
		return nil, fmt.Errorf("名称与内容不能为空")
	}
	if len(content) > 512*1024 {
		return nil, fmt.Errorf("脚本内容过长")
	}
	if req.GroupID != "" {
		if _, err := s.repo.FindGroup(req.GroupID); err != nil {
			return nil, fmt.Errorf("分组不存在")
		}
	}
	now := time.Now()
	if strings.TrimSpace(req.ID) == "" {
		sc := &Script{
			ID: uuid.NewString(), GroupID: req.GroupID, Name: name, Content: content,
			Interactive: req.Interactive, Description: strings.TrimSpace(req.Description),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := s.repo.Create(sc); err != nil {
			return nil, fmt.Errorf("创建失败")
		}
		return s.viewOf(sc)
	}
	sc, err := s.repo.Find(req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("脚本不存在")
		}
		return nil, err
	}
	sc.GroupID = req.GroupID
	sc.Name = name
	sc.Content = content
	sc.Interactive = req.Interactive
	sc.Description = strings.TrimSpace(req.Description)
	sc.UpdatedAt = now
	if err := s.repo.Update(sc); err != nil {
		return nil, fmt.Errorf("更新失败")
	}
	return s.viewOf(sc)
}

/**
 * Remove 批量删除脚本。
 */
func (s *Service) Remove(ids []string) error {
	clean := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		clean = append(clean, id)
	}
	if len(clean) == 0 {
		return fmt.Errorf("未选择脚本")
	}
	return s.repo.Delete(clean)
}

/**
 * Run 执行脚本：交互式返回正文供前端注入；非交互在本机或 SSH 主机跑完返回输出。
 */
func (s *Service) Run(req RunRequest) (*RunResult, error) {
	sc, err := s.repo.Find(req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("脚本不存在")
		}
		return nil, err
	}
	// 1. 交互式：前端开终端后注入
	if sc.Interactive {
		return &RunResult{
			Interactive: true,
			Message:     "请在终端中打开会话后注入执行",
			Content:     sc.Content,
		}, nil
	}
	// 2. 非交互：本机或远程一次性执行
	if strings.TrimSpace(req.HostID) == "" {
		return s.runLocal(sc.Content)
	}
	return s.runSSH(req.HostID, sc.Content)
}

func (s *Service) runLocal(content string) (*RunResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", content)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", content)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	res := &RunResult{
		Interactive: false,
		Stdout:      truncate(stdout.String(), outputLimit),
		Stderr:      truncate(stderr.String(), outputLimit),
		DurationMs:  time.Since(start).Milliseconds(),
	}
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			res.ExitCode = -1
			if res.Stderr == "" {
				res.Stderr = "执行超时"
			}
			return res, nil
		}
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		return nil, fmt.Errorf("执行失败: %v", err)
	}
	return res, nil
}

func (s *Service) runSSH(hostID, content string) (*RunResult, error) {
	if s.hosts == nil {
		return nil, fmt.Errorf("主机服务未就绪")
	}
	h, secret, pass, err := s.hosts.DecryptAuth(hostID)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	stdout, stderr, code, err := platterm.SSHRun(platterm.SSHRunOpts{
		Host:       h.Host,
		Port:       h.Port,
		Username:   h.Username,
		AuthType:   h.AuthType,
		Secret:     secret,
		Passphrase: pass,
		Command:    content,
		Timeout:    runTimeout,
	})
	res := &RunResult{
		Interactive: false,
		ExitCode:    code,
		Stdout:      truncate(stdout, outputLimit),
		Stderr:      truncate(stderr, outputLimit),
		DurationMs:  time.Since(start).Milliseconds(),
	}
	if err != nil && stdout == "" && stderr == "" {
		return nil, err
	}
	return res, nil
}

func (s *Service) viewOf(sc *Script) (*ScriptView, error) {
	name := ""
	if sc.GroupID != "" {
		m, err := s.repo.MapGroupNames([]string{sc.GroupID})
		if err != nil {
			return nil, err
		}
		name = m[sc.GroupID]
	}
	return &ScriptView{Script: *sc, GroupName: name}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
