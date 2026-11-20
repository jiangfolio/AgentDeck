package user

import (
	"errors"

	"gorm.io/gorm"
)

// Repo 用户仓储。
type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// FindByUsername 按用户名查用户。
func (r *Repo) FindByUsername(username string) (*User, error) {
	if username == "" {
		return nil, errors.New("用户名为空")
	}
	var u User
	err := r.db.Select("id", "username", "password_hash", "salt", "display_name", "created_at", "updated_at").
		Where("username = ?", username).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByID 按主键查用户。
func (r *Repo) FindByID(id int64) (*User, error) {
	if id <= 0 {
		return nil, nil
	}
	var u User
	err := r.db.Select("id", "username", "password_hash", "salt", "display_name", "created_at", "updated_at").
		Where("id = ?", id).
		First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Count 统计用户数。
func (r *Repo) Count() (int64, error) {
	var n int64
	err := r.db.Model(&User{}).Count(&n).Error
	return n, err
}

// Create 创建用户。
func (r *Repo) Create(u *User) error {
	return r.db.Create(u).Error
}
