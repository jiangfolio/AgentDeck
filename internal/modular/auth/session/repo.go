package session

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Repo 会话仓储。
type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// Create 写入会话。
func (r *Repo) Create(s *Session) error {
	return r.db.Create(s).Error
}

// FindValidByToken 按 token 查未过期会话。
func (r *Repo) FindValidByToken(token string) (*Session, error) {
	if token == "" {
		return nil, nil
	}
	var s Session
	err := r.db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteByToken 按 token 删除会话（登出）。
func (r *Repo) DeleteByToken(token string) error {
	if token == "" {
		return nil
	}
	return r.db.Where("token = ?", token).Delete(&Session{}).Error
}

// DeleteExpired 清理过期会话。
func (r *Repo) DeleteExpired() error {
	return r.db.Where("expires_at <= ?", time.Now()).Delete(&Session{}).Error
}
