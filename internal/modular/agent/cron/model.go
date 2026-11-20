package cron

import "time"

// Job 定时任务定义（落库）。
type Job struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	Name      string    `gorm:"size:128;not null;index" json:"name"`
	Expr      string    `gorm:"size:64;not null" json:"expr"`
	Command   string    `gorm:"size:2048;not null" json:"command"`
	WorkDir   string    `gorm:"size:512" json:"work_dir"`
	Enabled   bool      `gorm:"not null;index;default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Job) TableName() string { return "agent_cron_job" }

// ListQuery 任务列表查询。
type ListQuery struct {
	Page     int   `form:"page" binding:"required,min=1"`
	PageSize int   `form:"page_size" binding:"required,min=1,max=100"`
	Enabled  *bool `form:"enabled"`
}

// ListResult 分页结果。
type ListResult struct {
	Items    []JobView `json:"items"`
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
}

// JobView 列表/详情展示（含下次运行时间）。
type JobView struct {
	Job
	NextRunAt int64 `json:"next_run_at"`
}

// SaveRequest 创建或更新任务。
type SaveRequest struct {
	ID      string `json:"id"`
	Name    string `json:"name" binding:"required,max=128"`
	Expr    string `json:"expr" binding:"required,max=64"`
	Command string `json:"command" binding:"required,max=2048"`
	WorkDir string `json:"work_dir" binding:"max=512"`
	Enabled *bool  `json:"enabled"`
}

// IDRequest 仅含任务 ID。
type IDRequest struct {
	ID string `json:"id" binding:"required"`
}

// SetEnabledRequest 启停任务。
type SetEnabledRequest struct {
	ID      string `json:"id" binding:"required"`
	Enabled bool   `json:"enabled"`
}

// RunsQuery 运行记录查询。
type RunsQuery struct {
	ID string `form:"id" binding:"required"`
}

// RunRecord 单次运行记录。
type RunRecord struct {
	Ts         string `json:"ts"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
}
