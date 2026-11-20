package startup

import "gorm.io/gorm"

// Repo 自启条目仓储。
type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(e *Entry) error {
	return r.db.Create(e).Error
}

func (r *Repo) Update(e *Entry) error {
	return r.db.Model(e).Select("name", "command", "work_dir", "enabled", "platform_ref", "updated_at").Updates(e).Error
}

func (r *Repo) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&Entry{}).Error
}

func (r *Repo) FindByID(id string) (*Entry, error) {
	var e Entry
	err := r.db.Select("id", "name", "command", "work_dir", "enabled", "platform_ref", "created_at", "updated_at").
		Where("id = ?", id).First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *Repo) ListAll() ([]Entry, error) {
	var items []Entry
	err := r.db.Select("id", "name", "command", "work_dir", "enabled", "platform_ref", "created_at", "updated_at").
		Order("updated_at desc").Find(&items).Error
	return items, err
}
