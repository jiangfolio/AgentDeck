package termhost

import (
	"strings"

	"gorm.io/gorm"
)

// Repo 主机与分组仓储。
type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CreateGroup(g *HostGroup) error {
	return r.db.Create(g).Error
}

func (r *Repo) UpdateGroup(g *HostGroup) error {
	return r.db.Model(g).Select("name", "sort").Updates(g).Error
}

func (r *Repo) DeleteGroup(id string) error {
	return r.db.Where("id = ?", id).Delete(&HostGroup{}).Error
}

func (r *Repo) FindGroup(id string) (*HostGroup, error) {
	var g HostGroup
	err := r.db.Select("id", "name", "sort", "created_at").Where("id = ?", id).First(&g).Error
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *Repo) ListGroups() ([]HostGroup, error) {
	var items []HostGroup
	err := r.db.Select("id", "name", "sort", "created_at").Order("sort asc, created_at asc").Find(&items).Error
	return items, err
}

func (r *Repo) CreateHost(h *Host) error {
	return r.db.Create(h).Error
}

func (r *Repo) UpdateHost(h *Host, cols []string) error {
	return r.db.Model(h).Select(cols).Updates(h).Error
}

func (r *Repo) DeleteHosts(ids []string) error {
	return r.db.Where("id IN ?", ids).Delete(&Host{}).Error
}

func (r *Repo) FindHost(id string) (*Host, error) {
	var h Host
	err := r.db.Where("id = ?", id).First(&h).Error
	if err != nil {
		return nil, err
	}
	return &h, nil
}

/**
 * ListHosts 分页查询主机；keyword / groupID 可选。
 */
func (r *Repo) ListHosts(page, pageSize int, keyword, groupID string) ([]Host, int64, error) {
	q := r.db.Model(&Host{})
	if groupID != "" {
		q = q.Where("group_id = ?", groupID)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("title LIKE ? OR host LIKE ? OR username LIKE ? OR description LIKE ?", like, like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Host
	err := q.Select("id", "group_id", "title", "host", "port", "username", "auth_type", "secret_enc", "passphrase_enc", "description", "sort", "created_at", "updated_at").
		Order("sort asc, updated_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

func (r *Repo) CountHostsByGroup(groupID string) (int64, error) {
	var n int64
	err := r.db.Model(&Host{}).Where("group_id = ?", groupID).Count(&n).Error
	return n, err
}

/**
 * MapGroupNames 按 ID 批量取分组名，避免 N+1。
 */
func (r *Repo) MapGroupNames(ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	uniq := uniqueNonEmpty(ids)
	for i := 0; i < len(uniq); i += 500 {
		end := i + 500
		if end > len(uniq) {
			end = len(uniq)
		}
		batch := uniq[i:end]
		var groups []HostGroup
		if err := r.db.Select("id", "name").Where("id IN ?", batch).Find(&groups).Error; err != nil {
			return nil, err
		}
		for _, g := range groups {
			out[g.ID] = g.Name
		}
	}
	return out, nil
}

func uniqueNonEmpty(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
