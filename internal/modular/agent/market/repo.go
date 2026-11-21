package market

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repo 应用目录与运行状态仓储。
type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

/**
 * ListCatalog 分页查询目录；keyword/category 可选。
 */
func (r *Repo) ListCatalog(page, pageSize int, category, keyword string) ([]CatalogApp, int64, error) {
	q := r.db.Model(&CatalogApp{})
	if category != "" && category != "全部" {
		q = q.Where("category = ?", category)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		q = q.Where("id LIKE ? OR name LIKE ? OR summary LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []CatalogApp
	err := q.Select("id", "name", "summary", "category", "bin", "start", "env", "runnable", "sort", "created_at", "updated_at").
		Order("sort asc, updated_at desc").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

/**
 * DistinctCategories 返回目录中已有分类。
 */
func (r *Repo) DistinctCategories() ([]string, error) {
	var names []string
	err := r.db.Model(&CatalogApp{}).
		Where("category <> ''").
		Distinct().
		Order("category asc").
		Pluck("category", &names).Error
	return names, err
}

func (r *Repo) FindCatalog(id string) (*CatalogApp, error) {
	var item CatalogApp
	err := r.db.Select("id", "name", "summary", "category", "bin", "start", "env", "runnable", "sort", "created_at", "updated_at").
		Where("id = ?", id).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repo) CreateCatalog(item *CatalogApp) error {
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	return r.db.Create(item).Error
}

func (r *Repo) UpdateCatalog(item *CatalogApp) error {
	item.UpdatedAt = time.Now()
	return r.db.Model(item).Select(
		"name", "summary", "category", "bin", "start", "env", "runnable", "sort", "updated_at",
	).Updates(item).Error
}

func (r *Repo) DeleteCatalog(id string) error {
	return r.db.Where("id = ?", id).Delete(&CatalogApp{}).Error
}

func (r *Repo) CountCatalog() (int64, error) {
	var n int64
	err := r.db.Model(&CatalogApp{}).Count(&n).Error
	return n, err
}

/**
 * ListRuntimes 一次取出全部运行状态，禁止循环查库。
 */
func (r *Repo) ListRuntimes() ([]Runtime, error) {
	var items []Runtime
	err := r.db.Select("app_id", "pid", "updated_at").Find(&items).Error
	return items, err
}

func (r *Repo) GetRuntime(appID string) (*Runtime, error) {
	var item Runtime
	err := r.db.Select("app_id", "pid", "updated_at").Where("app_id = ?", appID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

/**
 * UpsertPid 写入或更新进程号。
 */
func (r *Repo) UpsertPid(appID string, pid int) error {
	item := Runtime{AppID: appID, Pid: pid, UpdatedAt: time.Now()}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "app_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"pid", "updated_at"}),
	}).Create(&item).Error
}

func (r *Repo) DeleteRuntime(appID string) error {
	return r.db.Where("app_id = ?", appID).Delete(&Runtime{}).Error
}
