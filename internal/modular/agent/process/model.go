package process

// ListQuery 进程列表筛选（全量返回，不分页）。
type ListQuery struct {
	Keyword string `form:"keyword"`
	User    string `form:"user"`
}

// ListResult 全量进程列表。
type ListResult struct {
	Items []Item `json:"items"`
	Total int    `json:"total"`
}

// Item 进程列表行。
type Item struct {
	PID        int32   `json:"pid"`
	PPID       int32   `json:"ppid"`
	Name       string  `json:"name"`
	Username   string  `json:"username"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float32 `json:"mem_percent"`
	Status     string  `json:"status"`
	Cmdline    string  `json:"cmdline"`
}

// DetailQuery 进程详情查询。
type DetailQuery struct {
	PID int32 `form:"pid" binding:"required,min=1"`
}

// Detail 进程详情。
type Detail struct {
	Item
	CreateTime  int64    `json:"create_time"`
	NumThreads  int32    `json:"num_threads"`
	MemoryRSS   uint64   `json:"memory_rss"`
	MemoryVMS   uint64   `json:"memory_vms"`
	Cwd         string   `json:"cwd"`
	Exe         string   `json:"exe"`
	CmdlineArgs []string `json:"cmdline_args"`
}

// KillRequest 结束进程请求。
type KillRequest struct {
	PID         int32 `json:"pid" binding:"required,min=1"`
	Tree        bool  `json:"tree"`
	Force       bool  `json:"force"`
	ConfirmSelf bool  `json:"confirm_self"`
}
