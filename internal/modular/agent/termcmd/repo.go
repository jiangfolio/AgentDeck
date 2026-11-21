package termcmd

import (
	"strings"

	"gorm.io/gorm"
)

// Repo 快速命令仓储。
type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(c *QuickCmd) error {
	return r.db.Create(c).Error
}

func (r *Repo) Update(c *QuickCmd) error {
	return r.db.Model(c).Select("group_name", "name", "command", "sort", "updated_at").Updates(c).Error
}

func (r *Repo) Delete(ids []string) error {
	return r.db.Where("id IN ?", ids).Delete(&QuickCmd{}).Error
}

func (r *Repo) Find(id string) (*QuickCmd, error) {
	var c QuickCmd
	err := r.db.Select("id", "group_name", "name", "command", "sort", "created_at", "updated_at").
		Where("id = ?", id).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

/**
 * List 分页查询；keyword / groupName 可选。
 */
func (r *Repo) List(page, pageSize int, keyword, groupName string) ([]QuickCmd, int64, error) {
	q := r.db.Model(&QuickCmd{})
	if groupName != "" {
		q = q.Where("group_name = ?", groupName)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("name LIKE ? OR command LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []QuickCmd
	err := q.Select("id", "group_name", "name", "command", "sort", "created_at", "updated_at").
		Order("sort asc, updated_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

/**
 * DistinctGroups 返回已使用的分组名（去重），供筛选下拉。
 */
func (r *Repo) DistinctGroups() ([]string, error) {
	var names []string
	err := r.db.Model(&QuickCmd{}).
		Where("group_name <> ''").
		Distinct().
		Order("group_name asc").
		Pluck("group_name", &names).Error
	return names, err
}
