package key

import "time"

// Key 持久化服务端 SM2 密钥对。
type Key struct {
	ID         int64     `gorm:"primaryKey"`
	Name       string    `gorm:"size:64;uniqueIndex;not null"`
	PrivateKey string    `gorm:"type:text;not null"`
	PublicKey  string    `gorm:"type:text;not null"`
	CreatedAt  time.Time
}

// TableName 固定表名，避免 GORM 默认复数化。
func (Key) TableName() string {
	return "crypto_keys"
}
