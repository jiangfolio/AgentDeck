package cron

import (
	"gorm.io/gorm"
)

// Repo 定时任务仓储。
type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(job *Job) error {
	return r.db.Create(job).Error
}

func (r *Repo) Update(job *Job) error {
	return r.db.Model(job).Select("name", "expr", "command", "work_dir", "enabled", "updated_at").Updates(job).Error
}

func (r *Repo) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&Job{}).Error
}

func (r *Repo) FindByID(id string) (*Job, error) {
	var job Job
	err := r.db.Select("id", "name", "expr", "command", "work_dir", "enabled", "created_at", "updated_at").
		Where("id = ?", id).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

/**
 * List 分页查询；enabled 为空则不过滤。
 */
func (r *Repo) List(page, pageSize int, enabled *bool) ([]Job, int64, error) {
	q := r.db.Model(&Job{})
	if enabled != nil {
		q = q.Where("enabled = ?", *enabled)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Job
	err := q.Select("id", "name", "expr", "command", "work_dir", "enabled", "created_at", "updated_at").
		Order("updated_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

func (r *Repo) ListAll() ([]Job, error) {
	var items []Job
	err := r.db.Select("id", "name", "expr", "command", "work_dir", "enabled", "created_at", "updated_at").
		Find(&items).Error
	return items, err
}
