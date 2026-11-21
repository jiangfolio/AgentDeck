package script

import (
	"strings"

	"gorm.io/gorm"
)

// Repo 脚本仓储。
type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CreateGroup(g *Group) error {
	return r.db.Create(g).Error
}

func (r *Repo) UpdateGroup(g *Group) error {
	return r.db.Model(g).Select("name", "sort").Updates(g).Error
}

func (r *Repo) DeleteGroup(id string) error {
	return r.db.Where("id = ?", id).Delete(&Group{}).Error
}

func (r *Repo) FindGroup(id string) (*Group, error) {
	var g Group
	err := r.db.Select("id", "name", "sort", "created_at").Where("id = ?", id).First(&g).Error
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *Repo) ListGroups() ([]Group, error) {
	var items []Group
	err := r.db.Select("id", "name", "sort", "created_at").Order("sort asc, created_at asc").Find(&items).Error
	return items, err
}

func (r *Repo) CountByGroup(groupID string) (int64, error) {
	var n int64
	err := r.db.Model(&Script{}).Where("group_id = ?", groupID).Count(&n).Error
	return n, err
}

func (r *Repo) Create(s *Script) error {
	return r.db.Create(s).Error
}

func (r *Repo) Update(s *Script) error {
	return r.db.Model(s).Select("group_id", "name", "content", "interactive", "description", "updated_at").Updates(s).Error
}

func (r *Repo) Delete(ids []string) error {
	return r.db.Where("id IN ?", ids).Delete(&Script{}).Error
}

func (r *Repo) Find(id string) (*Script, error) {
	var s Script
	err := r.db.Where("id = ?", id).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

/**
 * List 分页查询脚本。
 */
func (r *Repo) List(page, pageSize int, keyword, groupID string) ([]Script, int64, error) {
	q := r.db.Model(&Script{})
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Script
	err := q.Select("id", "group_id", "name", "content", "interactive", "description", "created_at", "updated_at").
		Order("updated_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

/**
 * MapGroupNames 批量取分组名。
 */
func (r *Repo) MapGroupNames(ids []string) (map[string]string, error) {
	out := map[string]string{}
	uniq := make([]string, 0, len(ids))
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
		uniq = append(uniq, id)
	}
	for i := 0; i < len(uniq); i += 500 {
		end := i + 500
		if end > len(uniq) {
			end = len(uniq)
		}
		var groups []Group
		if err := r.db.Select("id", "name").Where("id IN ?", uniq[i:end]).Find(&groups).Error; err != nil {
			return nil, err
		}
		for _, g := range groups {
			out[g.ID] = g.Name
		}
	}
	return out, nil
}
