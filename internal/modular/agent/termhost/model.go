package termhost

import "time"

// HostGroup 终端主机分组。
type HostGroup struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Name      string    `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Sort      int       `gorm:"not null;default:0;index" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
}

func (HostGroup) TableName() string { return "term_host_group" }

// Host SSH 主机档案（密钥密文落库）。
type Host struct {
	ID            string    `gorm:"primaryKey;size:36" json:"id"`
	GroupID       string    `gorm:"size:36;index" json:"group_id"`
	Title         string    `gorm:"size:128;not null;index" json:"title"`
	Host          string    `gorm:"size:255;not null;index" json:"host"`
	Port          int       `gorm:"not null;default:22" json:"port"`
	Username      string    `gorm:"size:128;not null" json:"username"`
	AuthType      string    `gorm:"size:32;not null" json:"auth_type"` // password | private_key
	SecretEnc     string    `gorm:"type:text" json:"-"`
	PassphraseEnc string    `gorm:"type:text" json:"-"`
	Description   string    `gorm:"size:512" json:"description"`
	Sort          int       `gorm:"not null;default:0;index" json:"sort"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Host) TableName() string { return "term_host" }

// HostView 对外展示，不含明文密钥。
type HostView struct {
	ID          string    `json:"id"`
	GroupID     string    `json:"group_id"`
	GroupName   string    `json:"group_name"`
	Title       string    `json:"title"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	Username    string    `json:"username"`
	AuthType    string    `json:"auth_type"`
	HasSecret   bool      `json:"has_secret"`
	Description string    `json:"description"`
	Sort        int       `json:"sort"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListQuery 主机列表查询。
type ListQuery struct {
	Page     int    `form:"page" binding:"required,min=1"`
	PageSize int    `form:"page_size" binding:"required,min=1,max=100"`
	Keyword  string `form:"keyword"`
	GroupID  string `form:"group_id"`
}

// ListResult 分页结果。
type ListResult struct {
	Items    []HostView `json:"items"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

// SaveHostRequest 创建或更新主机。
type SaveHostRequest struct {
	ID          string `json:"id"`
	GroupID     string `json:"group_id"`
	Title       string `json:"title" binding:"required,max=128"`
	Host        string `json:"host" binding:"required,max=255"`
	Port        int    `json:"port" binding:"required,min=1,max=65535"`
	Username    string `json:"username" binding:"required,max=128"`
	AuthType    string `json:"auth_type" binding:"required,oneof=password private_key"`
	Secret      string `json:"secret"`     // 密码或私钥；更新时空串表示不改
	Passphrase  string `json:"passphrase"` // 私钥口令；更新时空串表示不改
	Description string `json:"description" binding:"max=512"`
	Sort        int    `json:"sort"`
}

// SaveGroupRequest 创建或更新分组。
type SaveGroupRequest struct {
	ID   string `json:"id"`
	Name string `json:"name" binding:"required,max=128"`
	Sort int    `json:"sort"`
}

// IDRequest 单 ID。
type IDRequest struct {
	ID string `json:"id" binding:"required"`
}

// IDsRequest 批量 ID。
type IDsRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=100"`
}

// TestRequest 保存前探测连通性。secret 为空且带 id 时使用已存凭证。
type TestRequest struct {
	ID         string `json:"id"`
	Host       string `json:"host" binding:"required,max=255"`
	Port       int    `json:"port" binding:"required,min=1,max=65535"`
	Username   string `json:"username" binding:"required,max=128"`
	AuthType   string `json:"auth_type" binding:"required,oneof=password private_key"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// TestResult 探测结果，不含任何凭证。
type TestResult struct {
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latency_ms"`
	Message   string `json:"message"`
}
