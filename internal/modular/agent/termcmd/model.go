package termcmd

import "time"

// QuickCmd 终端快速命令。
type QuickCmd struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	GroupName string    `gorm:"size:128;index" json:"group_name"`
	Name      string    `gorm:"size:128;not null;index" json:"name"`
	Command   string    `gorm:"size:2048;not null" json:"command"`
	Sort      int       `gorm:"not null;default:0;index" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (QuickCmd) TableName() string { return "term_quick_cmd" }

// ListQuery 列表查询。
type ListQuery struct {
	Page      int    `form:"page" binding:"required,min=1"`
	PageSize  int    `form:"page_size" binding:"required,min=1,max=100"`
	Keyword   string `form:"keyword"`
	GroupName string `form:"group_name"`
}

// ListResult 分页结果。
type ListResult struct {
	Items    []QuickCmd `json:"items"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

// SaveRequest 创建或更新。
type SaveRequest struct {
	ID        string `json:"id"`
	GroupName string `json:"group_name" binding:"max=128"`
	Name      string `json:"name" binding:"required,max=128"`
	Command   string `json:"command" binding:"required,max=2048"`
	Sort      int    `json:"sort"`
}

// IDRequest 单 ID。
type IDRequest struct {
	ID string `json:"id" binding:"required"`
}

// IDsRequest 批量删除。
type IDsRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=100"`
}
