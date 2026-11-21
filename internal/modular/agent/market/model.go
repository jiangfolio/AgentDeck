package market

import (
	"time"

	"github.com/jiangbyte/agentdeck/internal/pkg/platform/apps"
)

// CatalogApp 可创建/编辑的应用目录项（SQLite）。
type CatalogApp struct {
	ID        string            `gorm:"primaryKey;size:64" json:"id"`
	Name      string            `gorm:"size:128;not null;index" json:"name"`
	Summary   string            `gorm:"size:512" json:"summary"`
	Category  string            `gorm:"size:64;index" json:"category"`
	Bin       string            `gorm:"size:256;not null" json:"bin"`
	Start     []string          `gorm:"serializer:json" json:"start"`
	Env       map[string]string `gorm:"serializer:json" json:"env"`
	Runnable  bool              `gorm:"not null;default:true;index" json:"runnable"`
	Sort      int               `gorm:"not null;default:0;index" json:"sort"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

func (CatalogApp) TableName() string { return "agent_market_app" }

// Layout 转为运行框架使用的 layout。
func (a CatalogApp) Layout() apps.Layout {
	runnable := a.Runnable
	return apps.Layout{
		Bin:      a.Bin,
		Env:      a.Env,
		Start:    a.Start,
		Runnable: &runnable,
	}
}

// Runtime 应用进程运行状态（一应用一行）。
type Runtime struct {
	AppID     string    `gorm:"primaryKey;size:64" json:"app_id"`
	Pid       int       `gorm:"not null;default:0" json:"pid"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Runtime) TableName() string { return "agent_market_runtime" }

// CatalogQuery 应用目录筛选。
type CatalogQuery struct {
	Tab      string `form:"tab"`
	Category string `form:"category"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// App 目录展示。
type App struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Summary string            `json:"summary"`
	Category string           `json:"category"`
	Bin     string            `json:"bin"`
	Start   []string          `json:"start"`
	Env     map[string]string `json:"env,omitempty"`
	Sort    int               `json:"sort"`
	AppDir  string            `json:"app_dir"`
	Running bool              `json:"running"`
	Pid     int               `json:"pid,omitempty"`
	Runnable bool             `json:"runnable"`
}

// CatalogResult 分页后的目录。
type CatalogResult struct {
	Items      []App    `json:"items"`
	Total      int64    `json:"total"`
	Page       int      `json:"page"`
	PageSize   int      `json:"page_size"`
	Categories []string `json:"categories"`
}

// SaveCatalogRequest 创建或更新目录项。
type SaveCatalogRequest struct {
	ID       string            `json:"id" binding:"required,max=64"`
	Name     string            `json:"name" binding:"required,max=128"`
	Summary  string            `json:"summary" binding:"max=512"`
	Category string            `json:"category" binding:"max=64"`
	Bin      string            `json:"bin" binding:"required,max=256"`
	Start    []string          `json:"start"`
	Env      map[string]string `json:"env"`
	Runnable bool              `json:"runnable"`
	Sort     int               `json:"sort"`
}

// ActionRequest 启停/删除请求。
type ActionRequest struct {
	ID string `json:"id" binding:"required"`
}

// UploadResult 上传官方文件后的结果。
type UploadResult struct {
	AppDir  string   `json:"app_dir"`
	Files   []string `json:"files"`
}

// StatusResult 单应用运行状态。
type StatusResult struct {
	ID      string `json:"id"`
	Running bool   `json:"running"`
	Pid     int    `json:"pid,omitempty"`
	AppDir  string `json:"app_dir"`
}
