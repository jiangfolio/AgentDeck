package startup

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	platstart "github.com/jiangbyte/agentdeck/internal/pkg/platform/startup"
	"gorm.io/gorm"
)

// Service 开机自启业务。
type Service struct {
	repo *Repo
	mgr  platstart.Manager
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo, mgr: platstart.New()}
}

/**
 * Reconcile 将库中启用的自定义条目同步到 OS。
 */
func (s *Service) Reconcile() error {
	entries, err := s.repo.ListAll()
	if err != nil {
		return err
	}
	for _, e := range entries {
		spec := platstart.EntrySpec{
			ID: e.ID, Name: e.Name, Command: e.Command,
			WorkDir: e.WorkDir, Enabled: e.Enabled,
		}
		if err := s.mgr.ApplyEntry(spec); err != nil {
			return err
		}
	}
	return nil
}

/**
 * Self 返回自身自启状态。
 */
func (s *Service) Self() (*SelfStatus, error) {
	enabled, err := s.mgr.SelfEnabled()
	if err != nil {
		return nil, err
	}
	exe, _ := platstart.ExecutablePath()
	return &SelfStatus{Enabled: enabled, ExePath: exe}, nil
}

/**
 * SetSelf 安装或卸载 AgentDeck 自身用户级自启。
 */
func (s *Service) SetSelf(enabled bool) error {
	return s.mgr.SetSelfEnabled(enabled)
}

/**
 * List 合并自身状态、自定义条目与平台扫描项后分页。
 */
func (s *Service) List(q ListQuery) (*ListResult, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}

	// 1. 自身一行
	items := make([]ListItem, 0, 32)
	self, _ := s.Self()
	exe := ""
	enabled := false
	if self != nil {
		exe = self.ExePath
		enabled = self.Enabled
	}
	items = append(items, ListItem{
		ID: "self", Name: "AgentDeck", Command: exe,
		Enabled: enabled, Managed: true, Source: "self",
		ReadOnly: false, Kind: "self",
	})

	// 2. 自定义条目
	customs, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}
	customIDs := map[string]struct{}{}
	for _, e := range customs {
		customIDs[e.ID] = struct{}{}
		items = append(items, ListItem{
			ID: e.ID, Name: e.Name, Command: e.Command, WorkDir: e.WorkDir,
			Enabled: e.Enabled, Managed: true, Source: "custom",
			ReadOnly: false, Kind: "custom",
		})
	}

	// 3. 平台扫描：跳过已在自定义中的 managed 项与 self
	platform, err := s.mgr.ListPlatformEntries()
	if err == nil {
		for _, p := range platform {
			if p.ID == "agentdeck" || p.ID == "AgentDeck-Self" || p.ID == "com.agentdeck.self" {
				continue
			}
			rawID := strings.TrimPrefix(p.ID, "agentdeck-")
			rawID = strings.TrimPrefix(rawID, "com.agentdeck.")
			rawID = strings.TrimPrefix(rawID, "AgentDeck-Startup-")
			if _, ok := customIDs[rawID]; ok {
				continue
			}
			if _, ok := customIDs[p.ID]; ok {
				continue
			}
			items = append(items, ListItem{
				ID: p.ID, Name: p.Name, Command: p.Command,
				Enabled: p.Enabled, Managed: p.Managed, Source: p.Source,
				ReadOnly: p.ReadOnly, Kind: "platform",
			})
		}
	}

	total := len(items)
	start := (q.Page - 1) * q.PageSize
	if start > total {
		start = total
	}
	end := start + q.PageSize
	if end > total {
		end = total
	}
	return &ListResult{
		Items: items[start:end], Total: total, Page: q.Page, PageSize: q.PageSize,
	}, nil
}

/**
 * Save 创建或更新自定义条目并同步 OS。
 */
func (s *Service) Save(req SaveRequest) (*Entry, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var e *Entry
	var err error
	if strings.TrimSpace(req.ID) == "" {
		e = &Entry{
			ID: uuid.NewString(), Name: strings.TrimSpace(req.Name),
			Command: req.Command, WorkDir: req.WorkDir, Enabled: enabled,
		}
		e.PlatformRef = "agentdeck-" + e.ID
		if err = s.repo.Create(e); err != nil {
			return nil, fmt.Errorf("保存失败")
		}
	} else {
		e, err = s.repo.FindByID(req.ID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("条目不存在")
			}
			return nil, err
		}
		e.Name = strings.TrimSpace(req.Name)
		e.Command = req.Command
		e.WorkDir = req.WorkDir
		e.Enabled = enabled
		if err = s.repo.Update(e); err != nil {
			return nil, fmt.Errorf("更新失败")
		}
	}
	if err = s.mgr.ApplyEntry(platstart.EntrySpec{
		ID: e.ID, Name: e.Name, Command: e.Command, WorkDir: e.WorkDir, Enabled: e.Enabled,
	}); err != nil {
		return nil, fmt.Errorf("同步系统自启失败: %v", err)
	}
	return e, nil
}

/**
 * Remove 移除自定义条目及 OS 项。
 */
func (s *Service) Remove(id string) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("条目不存在")
		}
		return err
	}
	_ = s.mgr.RemoveEntry(id)
	return s.repo.Delete(id)
}

/**
 * SetEnabled 启停自定义或 managed 平台项。
 */
func (s *Service) SetEnabled(id string, enabled bool) error {
	if id == "self" {
		return s.SetSelf(enabled)
	}
	e, err := s.repo.FindByID(id)
	if err == nil {
		e.Enabled = enabled
		if err := s.repo.Update(e); err != nil {
			return fmt.Errorf("更新失败")
		}
		return s.mgr.ApplyEntry(platstart.EntrySpec{
			ID: e.ID, Name: e.Name, Command: e.Command, WorkDir: e.WorkDir, Enabled: e.Enabled,
		})
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	return s.mgr.SetEntryEnabled(id, enabled)
}
