package firewall

// Rule 一条入站防火墙规则。
type Rule struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	Protocol    string `json:"protocol"`
	IP          string `json:"ip"`
	Port        string `json:"port"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	Managed     bool   `json:"managed"`
}

// Status 防火墙后端、开关，以及未安装时能否在线或离线安装。
type Status struct {
	Backend    string   `json:"backend"`
	Installed  bool     `json:"installed"`
	Enabled    bool     `json:"enabled"`
	Version    string   `json:"version"`
	Message    string   `json:"message"`
	OS         string   `json:"os"`
	Manager    string   `json:"manager"`
	CanOnline  bool     `json:"can_online"`
	CanOffline bool     `json:"can_offline"`
	OfflineExt []string `json:"offline_ext"`
	Package    string   `json:"package"`
	IsRoot     bool     `json:"is_root"`
}

// Manager 跨平台防火墙读写。
type Manager interface {
	Status() Status
	Rules() ([]Rule, error)
	SetEnabled(enabled bool) error
	Save(rule Rule) error
	Remove(id string) error
	Reset() error
	InstallOnline() error
	InstallOffline(path string) error
}
