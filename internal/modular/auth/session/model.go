package session

import "time"

// Session 服务端会话（opaque token，非 JWT）。
type Session struct {
	ID        int64     `gorm:"primaryKey"`
	UserID    int64     `gorm:"index;not null"`
	Token     string    `gorm:"size:64;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time
}
