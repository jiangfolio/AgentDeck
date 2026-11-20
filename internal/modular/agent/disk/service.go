package disk

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	gdisk "github.com/shirou/gopsutil/v4/disk"
)

// Service 只读磁盘与分区。
type Service struct{}

func NewService() *Service {
	return &Service{}
}

/**
 * List 按平台把分区归到磁盘上，并标出系统盘。
 */
func (s *Service) List() *ListResult {
	// 1. 读取真实分区用量，丢掉伪文件系统
	parts := usablePartitions()
	if len(parts) == 0 {
		return &ListResult{Items: []Disk{}, Message: "未读取到磁盘分区"}
	}

	// 2. Linux 用 /sys/block 归组；其它系统按设备名前缀或盘符归组
	var disks []Disk
	switch runtime.GOOS {
	case "linux":
		disks = groupLinux(parts)
	case "windows":
		disks = groupWindows(parts)
	default:
		disks = groupDarwin(parts)
	}
	if disks == nil {
		disks = []Disk{}
	}
	return &ListResult{Items: disks}
}

func usablePartitions() []gdisk.PartitionStat {
	parts, err := gdisk.Partitions(false)
	if err != nil {
		return nil
	}
	out := make([]gdisk.PartitionStat, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		if skipFilesystem(p.Fstype, p.Mountpoint) {
			continue
		}
		key := p.Device + "|" + p.Mountpoint
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}

func toPartition(p gdisk.PartitionStat) (Partition, bool) {
	usage, err := gdisk.Usage(p.Mountpoint)
	if err != nil || usage == nil || usage.Total == 0 {
		return Partition{}, false
	}
	name := p.Device
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	return Partition{
		Name:         name,
		SizeBytes:    usage.Total,
		UsedBytes:    usage.Used,
		FreeBytes:    usage.Free,
		UsagePercent: round2(usage.UsedPercent),
		Mountpoint:   p.Mountpoint,
		Fstype:       p.Fstype,
	}, true
}

func groupLinux(parts []gdisk.PartitionStat) []Disk {
	// 1. 整盘容量来自 /sys/block，跳过 loop/ram
	sizes := map[string]uint64{}
	entries, err := os.ReadDir("/sys/block")
	if err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "fd") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join("/sys/block", name, "size"))
			if err != nil {
				continue
			}
			sectors, _ := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 64)
			sizes[name] = sectors * 512
		}
	}

	// 2. 分区按最长盘名前缀归组，根挂载所在盘标为系统盘
	buckets := map[string]*Disk{}
	for _, p := range parts {
		part, ok := toPartition(p)
		if !ok {
			continue
		}
		diskName := matchDisk(part.Name, sizes)
		d := buckets[diskName]
		if d == nil {
			d = &Disk{Name: diskName, SizeBytes: sizes[diskName], Partitions: []Partition{}}
			buckets[diskName] = d
		}
		if p.Mountpoint == "/" {
			d.System = true
		}
		d.Partitions = append(d.Partitions, part)
		if d.SizeBytes == 0 {
			d.SizeBytes += part.SizeBytes
		}
	}
	return flattenDisks(buckets)
}

func matchDisk(partName string, sizes map[string]uint64) string {
	best := ""
	for name := range sizes {
		if strings.HasPrefix(partName, name) && len(name) > len(best) {
			best = name
		}
	}
	if best != "" {
		return best
	}
	return trimPartSuffix(partName)
}

func groupDarwin(parts []gdisk.PartitionStat) []Disk {
	buckets := map[string]*Disk{}
	for _, p := range parts {
		part, ok := toPartition(p)
		if !ok {
			continue
		}
		name := trimDarwin(part.Name)
		d := buckets[name]
		if d == nil {
			d = &Disk{Name: name, Partitions: []Partition{}}
			buckets[name] = d
		}
		if p.Mountpoint == "/" {
			d.System = true
		}
		d.Partitions = append(d.Partitions, part)
		d.SizeBytes += part.SizeBytes
	}
	return flattenDisks(buckets)
}

func groupWindows(parts []gdisk.PartitionStat) []Disk {
	system := strings.ToUpper(os.Getenv("SystemDrive"))
	if system == "" {
		system = "C:"
	}
	buckets := map[string]*Disk{}
	for _, p := range parts {
		part, ok := toPartition(p)
		if !ok {
			continue
		}
		name := strings.ToUpper(part.Mountpoint)
		if len(name) >= 2 && name[1] == ':' {
			name = name[:2]
		}
		d := buckets[name]
		if d == nil {
			d = &Disk{Name: name, Partitions: []Partition{}, System: name == system}
			buckets[name] = d
		}
		d.Partitions = append(d.Partitions, part)
		d.SizeBytes += part.SizeBytes
	}
	return flattenDisks(buckets)
}

func flattenDisks(buckets map[string]*Disk) []Disk {
	names := make([]string, 0, len(buckets))
	for name := range buckets {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]Disk, 0, len(names))
	for _, name := range names {
		d := buckets[name]
		sort.Slice(d.Partitions, func(i, j int) bool { return d.Partitions[i].Name < d.Partitions[j].Name })
		out = append(out, *d)
	}
	return out
}

func trimPartSuffix(name string) string {
	// nvme0n1p1 -> nvme0n1，sda1 -> sda
	if i := strings.LastIndex(name, "p"); i > 0 && isDigits(name[i+1:]) && strings.Contains(name, "nvme") {
		return name[:i]
	}
	i := len(name)
	for i > 0 && name[i-1] >= '0' && name[i-1] <= '9' {
		i--
	}
	if i > 0 && i < len(name) {
		return name[:i]
	}
	return name
}

func trimDarwin(name string) string {
	// disk0s1 -> disk0
	if i := strings.Index(name, "s"); i > 0 && strings.HasPrefix(name, "disk") && isDigits(name[i+1:]) {
		return name[:i]
	}
	return trimPartSuffix(name)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func skipFilesystem(fstype, mountpoint string) bool {
	ft := strings.ToLower(fstype)
	mp := strings.ToLower(mountpoint)
	switch ft {
	case "tmpfs", "devtmpfs", "sysfs", "proc", "devfs", "overlay",
		"squashfs", "cgroup", "cgroup2", "pstore", "bpf", "tracefs",
		"securityfs", "debugfs", "fusectl", "mqueue", "hugetlbfs",
		"configfs", "binfmt_misc", "autofs", "nsfs", "ramfs":
		return true
	}
	if strings.HasPrefix(mp, "/sys") || strings.HasPrefix(mp, "/proc") ||
		strings.HasPrefix(mp, "/dev") || strings.HasPrefix(mp, "/run") {
		return true
	}
	return false
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
