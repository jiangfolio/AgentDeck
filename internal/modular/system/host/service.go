package host

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

// Service 采集跨平台主机与资源信息。
type Service struct{}

func NewService() *Service {
	return &Service{}
}

// Overview 汇总主机、CPU、内存、磁盘与进程运行时信息。
func (s *Service) Overview() (*Overview, error) {
	// 1. 主机 / OS
	hostInfo, err := collectHost()
	if err != nil {
		return nil, err
	}
	// 2. CPU
	cpuInfo, err := collectCPU()
	if err != nil {
		return nil, err
	}
	// 3. 内存与交换区
	memInfo, swapInfo, err := collectMemory()
	if err != nil {
		return nil, err
	}
	// 4. 磁盘分区（过滤伪文件系统）
	disks, err := collectDisks()
	if err != nil {
		return nil, err
	}
	// 5. 当前进程运行时
	return &Overview{
		Host:    *hostInfo,
		CPU:     *cpuInfo,
		Memory:  *memInfo,
		Swap:    *swapInfo,
		Disks:   disks,
		Runtime: collectRuntime(),
	}, nil
}

func collectHost() (*HostInfo, error) {
	info, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("读取主机信息失败: %w", err)
	}
	return &HostInfo{
		Hostname:        info.Hostname,
		OS:              info.OS,
		Platform:        info.Platform,
		PlatformVersion: info.PlatformVersion,
		KernelVersion:   info.KernelVersion,
		Arch:            info.KernelArch,
		UptimeSeconds:   info.Uptime,
	}, nil
}

func collectCPU() (*CPUInfo, error) {
	// 1. 逻辑核数与型号
	logical, err := cpu.Counts(true)
	if err != nil {
		return nil, fmt.Errorf("读取 CPU 逻辑核数失败: %w", err)
	}
	physical, err := cpu.Counts(false)
	if err != nil {
		physical = logical
	}
	modelName := ""
	infos, err := cpu.Info()
	if err == nil && len(infos) > 0 {
		modelName = strings.TrimSpace(infos[0].ModelName)
	}
	// 2. 短采样占用率（跨平台）
	percents, err := cpu.Percent(300*time.Millisecond, false)
	usage := 0.0
	if err == nil && len(percents) > 0 {
		usage = round2(percents[0])
	}
	return &CPUInfo{
		ModelName:     modelName,
		LogicalCores:  logical,
		PhysicalCores: physical,
		UsagePercent:  usage,
	}, nil
}

func collectMemory() (*MemoryInfo, *MemoryInfo, error) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return nil, nil, fmt.Errorf("读取内存失败: %w", err)
	}
	memInfo := &MemoryInfo{
		TotalBytes:     vm.Total,
		UsedBytes:      vm.Used,
		AvailableBytes: vm.Available,
		UsagePercent:   round2(vm.UsedPercent),
	}
	swapInfo := &MemoryInfo{}
	sm, err := mem.SwapMemory()
	if err == nil && sm != nil {
		swapInfo.TotalBytes = sm.Total
		swapInfo.UsedBytes = sm.Used
		swapInfo.AvailableBytes = sm.Free
		swapInfo.UsagePercent = round2(sm.UsedPercent)
	}
	return memInfo, swapInfo, nil
}

func collectDisks() ([]DiskInfo, error) {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil, fmt.Errorf("读取磁盘分区失败: %w", err)
	}
	seen := make(map[string]struct{})
	out := make([]DiskInfo, 0, len(parts))
	for _, p := range parts {
		if skipFilesystem(p.Fstype, p.Mountpoint) {
			continue
		}
		// 同一设备只保留一个挂载点，避免重复
		key := p.Device + "|" + p.Mountpoint
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil || usage == nil || usage.Total == 0 {
			continue
		}
		out = append(out, DiskInfo{
			Device:       p.Device,
			Mountpoint:   p.Mountpoint,
			Fstype:       p.Fstype,
			TotalBytes:   usage.Total,
			UsedBytes:    usage.Used,
			FreeBytes:    usage.Free,
			UsagePercent: round2(usage.UsedPercent),
		})
	}
	return out, nil
}

func collectRuntime() RuntimeInfo {
	return RuntimeInfo{
		GoVersion:  runtime.Version(),
		NumCPU:     runtime.NumCPU(),
		Goroutines: runtime.NumGoroutine(),
		PID:        os.Getpid(),
	}
}

func skipFilesystem(fstype, mountpoint string) bool {
	ft := strings.ToLower(fstype)
	mp := strings.ToLower(mountpoint)
	// 伪文件系统 / 临时挂载，各平台常见
	switch ft {
	case "tmpfs", "devtmpfs", "sysfs", "proc", "devfs", "overlay",
		"squashfs", "cgroup", "cgroup2", "pstore", "bpf", "tracefs",
		"securityfs", "debugfs", "fusectl", "mqueue", "hugetlbfs",
		"configfs", "binfmt_misc", "autofs", "nsfs", "ramfs":
		return true
	}
	if strings.HasPrefix(mp, "/sys") || strings.HasPrefix(mp, "/proc") ||
		strings.HasPrefix(mp, "/dev") || strings.HasPrefix(mp, "/run") ||
		strings.HasPrefix(mp, "/var/lib/docker") {
		return true
	}
	return false
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
