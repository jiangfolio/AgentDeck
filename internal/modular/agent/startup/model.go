package startup

import "time"

// Entry 用户自定义自启条目（落库）。
type Entry struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	Name        string    `gorm:"size:128;not null;index" json:"name"`
	Command     string    `gorm:"size:2048;not null" json:"command"`
	WorkDir     string    `gorm:"size:512" json:"work_dir"`
	Enabled     bool      `gorm:"not null;index;default:true" json:"enabled"`
	PlatformRef string    `gorm:"size:256" json:"platform_ref"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Entry) TableName() string { return "agent_startup_entry" }

// SelfStatus AgentDeck 自身自启状态。
type SelfStatus struct {
	Enabled bool   `json:"enabled"`
	ExePath string `json:"exe_path"`
}

// SetSelfRequest 设置自身自启。
type SetSelfRequest struct {
	Enabled bool `json:"enabled"`
}

// ListQuery 自启列表查询。
type ListQuery struct {
	Page     int `form:"page" binding:"required,min=1"`
	PageSize int `form:"page_size" binding:"required,min=1,max=100"`
}

// ListItem 列表项（自定义 + 平台扫描）。
type ListItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Command  string `json:"command"`
	WorkDir  string `json:"work_dir"`
	Enabled  bool   `json:"enabled"`
	Managed  bool   `json:"managed"`
	Source   string `json:"source"`
	ReadOnly bool   `json:"read_only"`
	Kind     string `json:"kind"` // custom | platform | self
}

// ListResult 分页结果。
type ListResult struct {
	Items    []ListItem `json:"items"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

// SaveRequest 保存自定义条目。
type SaveRequest struct {
	ID      string `json:"id"`
	Name    string `json:"name" binding:"required,max=128"`
	Command string `json:"command" binding:"required,max=2048"`
	WorkDir string `json:"work_dir" binding:"max=512"`
	Enabled *bool  `json:"enabled"`
}

// IDRequest 仅 ID。
type IDRequest struct {
	ID string `json:"id" binding:"required"`
}

// SetEnabledRequest 启停。
type SetEnabledRequest struct {
	ID      string `json:"id" binding:"required"`
	Enabled bool   `json:"enabled"`
}
