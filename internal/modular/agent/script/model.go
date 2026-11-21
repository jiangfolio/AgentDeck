package script

import "time"

// Group 脚本分组。
type Group struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Name      string    `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Sort      int       `gorm:"not null;default:0;index" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
}

func (Group) TableName() string { return "script_group" }

// Script 脚本库条目。
type Script struct {
	ID          string    `gorm:"primaryKey;size:36" json:"id"`
	GroupID     string    `gorm:"size:36;index" json:"group_id"`
	Name        string    `gorm:"size:128;not null;index" json:"name"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	Interactive bool      `gorm:"not null;default:false" json:"interactive"`
	Description string    `gorm:"size:512" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Script) TableName() string { return "script" }

// ScriptView 列表展示含分组名。
type ScriptView struct {
	Script
	GroupName string `json:"group_name"`
}

// ListQuery 脚本列表。
type ListQuery struct {
	Page     int    `form:"page" binding:"required,min=1"`
	PageSize int    `form:"page_size" binding:"required,min=1,max=100"`
	Keyword  string `form:"keyword"`
	GroupID  string `form:"group_id"`
}

// ListResult 分页结果。
type ListResult struct {
	Items    []ScriptView `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
}

// SaveScriptRequest 创建或更新脚本。
type SaveScriptRequest struct {
	ID          string `json:"id"`
	GroupID     string `json:"group_id"`
	Name        string `json:"name" binding:"required,max=128"`
	Content     string `json:"content" binding:"required"`
	Interactive bool   `json:"interactive"`
	Description string `json:"description" binding:"max=512"`
}

// SaveGroupRequest 分组保存。
type SaveGroupRequest struct {
	ID   string `json:"id"`
	Name string `json:"name" binding:"required,max=128"`
	Sort int    `json:"sort"`
}

// IDRequest 单 ID。
type IDRequest struct {
	ID string `json:"id" binding:"required"`
}

// IDsRequest 批量。
type IDsRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=100"`
}

// RunRequest 执行脚本。
type RunRequest struct {
	ID     string `json:"id" binding:"required"`
	HostID string `json:"host_id"` // 空=本机
}

// RunResult 非交互执行结果；交互式仅返回提示。
type RunResult struct {
	Interactive bool   `json:"interactive"`
	Message     string `json:"message,omitempty"`
	ExitCode    int    `json:"exit_code,omitempty"`
	Stdout      string `json:"stdout,omitempty"`
	Stderr      string `json:"stderr,omitempty"`
	DurationMs  int64  `json:"duration_ms,omitempty"`
	Content     string `json:"content,omitempty"` // 交互式：脚本正文供前端注入终端
}
