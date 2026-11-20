package key

import (
	"errors"

	"gorm.io/gorm"
)

// Repo SM2 密钥仓储。
type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// FindByName 按名称查密钥。
func (r *Repo) FindByName(name string) (*Key, error) {
	var row Key
	err := r.db.Where("name = ?", name).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// Create 写入密钥记录。
func (r *Repo) Create(row *Key) error {
	return r.db.Create(row).Error
}
