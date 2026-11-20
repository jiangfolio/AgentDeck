package monitor

// Snapshot 当前一拍监控数据（不含历史）。
type Snapshot struct {
	TS     int64     `json:"ts"`
	Load   LoadInfo  `json:"load"`
	CPU    CPUStat   `json:"cpu"`
	Memory MemStat   `json:"memory"`
	DiskIO DiskIO    `json:"disk_io"`
	Net    NetStat   `json:"net"`
	Top    []TopProc `json:"top"`
}

// LoadInfo 平均负载。Windows 上 supported 为 false。
type LoadInfo struct {
	Supported bool    `json:"supported"`
	Load1     float64 `json:"load1"`
	Load5     float64 `json:"load5"`
	Load15    float64 `json:"load15"`
}

// CPUStat CPU 总占用与每核占用。
type CPUStat struct {
	UsagePercent float64   `json:"usage_percent"`
	PerCore      []float64 `json:"per_core"`
}

// MemStat 内存用量。
type MemStat struct {
	TotalBytes     uint64  `json:"total_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
}

// DiskIO 磁盘读写速率。
type DiskIO struct {
	ReadBytesPerSec  float64        `json:"read_bytes_per_sec"`
	WriteBytesPerSec float64        `json:"write_bytes_per_sec"`
	Devices          []DiskIODevice `json:"devices"`
}

// DiskIODevice 单块设备的读写速率。
type DiskIODevice struct {
	Name             string  `json:"name"`
	ReadBytesPerSec  float64 `json:"read_bytes_per_sec"`
	WriteBytesPerSec float64 `json:"write_bytes_per_sec"`
}

// NetStat 网卡流量。
type NetStat struct {
	TotalSentBytes uint64     `json:"total_sent_bytes"`
	TotalRecvBytes uint64     `json:"total_recv_bytes"`
	Interfaces     []NetIface `json:"interfaces"`
}

// NetIface 单块网卡的上下行速率与累计字节。
type NetIface struct {
	Name            string  `json:"name"`
	BytesSentPerSec float64 `json:"bytes_sent_per_sec"`
	BytesRecvPerSec float64 `json:"bytes_recv_per_sec"`
	BytesSent       uint64  `json:"bytes_sent"`
	BytesRecv       uint64  `json:"bytes_recv"`
}

// TopProc 当前 CPU 占用最高的进程。
type TopProc struct {
	PID        int32   `json:"pid"`
	User       string  `json:"user"`
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpu_percent"`
}
