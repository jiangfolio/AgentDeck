package disk

// ListResult 磁盘与分区。
type ListResult struct {
	Items   []Disk `json:"items"`
	Message string `json:"message"`
}

// Disk 一块物理盘或逻辑盘。
type Disk struct {
	Name       string      `json:"name"`
	SizeBytes  uint64      `json:"size_bytes"`
	System     bool        `json:"system"`
	Partitions []Partition `json:"partitions"`
}

// Partition 分区用量。
type Partition struct {
	Name         string  `json:"name"`
	SizeBytes    uint64  `json:"size_bytes"`
	UsedBytes    uint64  `json:"used_bytes"`
	FreeBytes    uint64  `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
	Mountpoint   string  `json:"mountpoint"`
	Fstype       string  `json:"fstype"`
}
