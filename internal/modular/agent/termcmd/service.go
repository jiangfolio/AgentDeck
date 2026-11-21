package termcmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Service 快速命令业务。
type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

/**
 * List 分页列出快速命令。
 */
func (s *Service) List(q ListQuery) (*ListResult, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	items, total, err := s.repo.List(q.Page, q.PageSize, q.Keyword, q.GroupName)
	if err != nil {
		return nil, err
	}
	return &ListResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

/**
 * Groups 返回已有分组名列表。
 */
func (s *Service) Groups() ([]string, error) {
	return s.repo.DistinctGroups()
}

/**
 * Save 创建或更新快速命令。
 */
func (s *Service) Save(req SaveRequest) (*QuickCmd, error) {
	name := strings.TrimSpace(req.Name)
	cmd := strings.TrimSpace(req.Command)
	if name == "" || cmd == "" {
		return nil, fmt.Errorf("名称与命令不能为空")
	}
	now := time.Now()
	if strings.TrimSpace(req.ID) == "" {
		c := &QuickCmd{
			ID: uuid.NewString(), GroupName: strings.TrimSpace(req.GroupName), Name: name, Command: cmd,
			Sort: req.Sort, CreatedAt: now, UpdatedAt: now,
		}
		if err := s.repo.Create(c); err != nil {
			return nil, fmt.Errorf("创建失败")
		}
		return c, nil
	}
	c, err := s.repo.Find(req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("记录不存在")
		}
		return nil, err
	}
	c.GroupName = strings.TrimSpace(req.GroupName)
	c.Name = name
	c.Command = cmd
	c.Sort = req.Sort
	c.UpdatedAt = now
	if err := s.repo.Update(c); err != nil {
		return nil, fmt.Errorf("更新失败")
	}
	return c, nil
}

/**
 * Remove 批量删除。
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
		return fmt.Errorf("未选择记录")
	}
	return s.repo.Delete(clean)
}
