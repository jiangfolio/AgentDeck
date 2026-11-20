package firewall

import plat "github.com/jiangbyte/agentdeck/internal/pkg/platform/firewall"

// RulesQuery 规则列表筛选。
type RulesQuery struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// RulesResult 分页规则。
type RulesResult struct {
	Items    []plat.Rule `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Message  string      `json:"message"`
}

// SaveRequest 新建或更新规则。
type SaveRequest struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	Protocol    string `json:"protocol"`
	IP          string `json:"ip"`
	Port        string `json:"port"`
	Description string `json:"description"`
}

// IDRequest 按 id 删除。
type IDRequest struct {
	ID string `json:"id"`
}

// EnabledRequest 防火墙开关。
type EnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// ImportRequest 导入规则。mode 为 merge 或 replace。
type ImportRequest struct {
	Mode  string      `json:"mode"`
	Rules []plat.Rule `json:"rules"`
}
