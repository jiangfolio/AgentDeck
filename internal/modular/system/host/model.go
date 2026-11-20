package host

// Overview 主机系统与资源概览（跨平台：linux / windows / darwin）。
type Overview struct {
	Host    HostInfo    `json:"host"`
	CPU     CPUInfo     `json:"cpu"`
	Memory  MemoryInfo  `json:"memory"`
	Swap    MemoryInfo  `json:"swap"`
	Disks   []DiskInfo  `json:"disks"`
	Runtime RuntimeInfo `json:"runtime"`
}

// HostInfo 操作系统与主机信息。
type HostInfo struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platform_version"`
	KernelVersion   string `json:"kernel_version"`
	Arch            string `json:"arch"`
	UptimeSeconds   uint64 `json:"uptime_seconds"`
}

// CPUInfo CPU 型号与占用。
type CPUInfo struct {
	ModelName     string  `json:"model_name"`
	LogicalCores  int     `json:"logical_cores"`
	PhysicalCores int     `json:"physical_cores"`
	UsagePercent  float64 `json:"usage_percent"`
}

// MemoryInfo 内存或交换区用量。
type MemoryInfo struct {
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
}

// DiskInfo 磁盘分区用量。
type DiskInfo struct {
	Device       string  `json:"device"`
	Mountpoint   string  `json:"mountpoint"`
	Fstype       string  `json:"fstype"`
	TotalBytes   uint64  `json:"total_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
}

// RuntimeInfo 当前进程运行时信息。
type RuntimeInfo struct {
	GoVersion  string `json:"go_version"`
	NumCPU     int    `json:"num_cpu"`
	Goroutines int    `json:"goroutines"`
	PID        int    `json:"pid"`
}
