package monitor

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type diskCounter struct {
	read  uint64
	write uint64
}

type netCounter struct {
	sent uint64
	recv uint64
}

// Service 采集一拍监控，并用上一拍计数器计算速率。
type Service struct {
	mu       sync.Mutex
	hasPrev  bool
	prevAt   time.Time
	prevDisk map[string]diskCounter
	prevNet  map[string]netCounter
}

func NewService() *Service {
	return &Service{}
}

/**
 * Snapshot 采集当前负载、CPU、内存、磁盘 IO、网卡与 Top 进程。
 */
func (s *Service) Snapshot() (*Snapshot, error) {
	now := time.Now()
	snap := &Snapshot{TS: now.UnixMilli(), Top: []TopProc{}}

	// 1. 负载：Windows 无此指标，标记不支持而不是报错
	if avg, err := load.Avg(); err == nil && avg != nil {
		snap.Load = LoadInfo{Supported: true, Load1: round2(avg.Load1), Load5: round2(avg.Load5), Load15: round2(avg.Load15)}
	}

	// 2. CPU 每核占用，总数取平均，避免再采样一次
	if cores, err := cpu.Percent(200*time.Millisecond, true); err == nil && len(cores) > 0 {
		sum := 0.0
		snap.CPU.PerCore = make([]float64, len(cores))
		for i, v := range cores {
			snap.CPU.PerCore[i] = round2(v)
			sum += v
		}
		snap.CPU.UsagePercent = round2(sum / float64(len(cores)))
	}

	// 3. 内存
	if vm, err := mem.VirtualMemory(); err == nil && vm != nil {
		snap.Memory = MemStat{
			TotalBytes:     vm.Total,
			UsedBytes:      vm.Used,
			AvailableBytes: vm.Available,
			UsagePercent:   round2(vm.UsedPercent),
		}
	}

	// 4. 磁盘 IO / 网卡与上一拍做差分得到速率
	diskNow := readDisk()
	netNow := readNet()
	s.mu.Lock()
	sec := now.Sub(s.prevAt).Seconds()
	if s.hasPrev && sec > 0.2 {
		snap.DiskIO = diffDisk(s.prevDisk, diskNow, sec)
		snap.Net = diffNet(s.prevNet, netNow, sec)
	} else {
		snap.DiskIO = DiskIO{Devices: []DiskIODevice{}}
		snap.Net = totalsOnly(netNow)
	}
	s.prevDisk = diskNow
	s.prevNet = netNow
	s.prevAt = now
	s.hasPrev = true
	s.mu.Unlock()

	// 5. 当前 CPU 最高的 5 个进程
	snap.Top = topProcs(5)
	return snap, nil
}

func readDisk() map[string]diskCounter {
	out := map[string]diskCounter{}
	counters, err := disk.IOCounters()
	if err != nil {
		return out
	}
	for name, c := range counters {
		out[name] = diskCounter{read: c.ReadBytes, write: c.WriteBytes}
	}
	return out
}

func readNet() map[string]netCounter {
	out := map[string]netCounter{}
	counters, err := gnet.IOCounters(true)
	if err != nil {
		return out
	}
	for _, c := range counters {
		name := strings.ToLower(c.Name)
		if name == "lo" || strings.HasPrefix(name, "lo") {
			continue
		}
		out[c.Name] = netCounter{sent: c.BytesSent, recv: c.BytesRecv}
	}
	return out
}

func diffDisk(prev, cur map[string]diskCounter, sec float64) DiskIO {
	io := DiskIO{Devices: []DiskIODevice{}}
	names := make([]string, 0, len(cur))
	for name := range cur {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c := cur[name]
		p, ok := prev[name]
		if !ok {
			continue
		}
		rd := rate(c.read, p.read, sec)
		wr := rate(c.write, p.write, sec)
		io.ReadBytesPerSec += rd
		io.WriteBytesPerSec += wr
		io.Devices = append(io.Devices, DiskIODevice{Name: name, ReadBytesPerSec: round2(rd), WriteBytesPerSec: round2(wr)})
	}
	io.ReadBytesPerSec = round2(io.ReadBytesPerSec)
	io.WriteBytesPerSec = round2(io.WriteBytesPerSec)
	return io
}

func diffNet(prev, cur map[string]netCounter, sec float64) NetStat {
	st := NetStat{Interfaces: []NetIface{}}
	names := make([]string, 0, len(cur))
	for name := range cur {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c := cur[name]
		st.TotalSentBytes += c.sent
		st.TotalRecvBytes += c.recv
		p, ok := prev[name]
		sentRate, recvRate := 0.0, 0.0
		if ok {
			sentRate = rate(c.sent, p.sent, sec)
			recvRate = rate(c.recv, p.recv, sec)
		}
		st.Interfaces = append(st.Interfaces, NetIface{
			Name:            name,
			BytesSentPerSec: round2(sentRate),
			BytesRecvPerSec: round2(recvRate),
			BytesSent:       c.sent,
			BytesRecv:       c.recv,
		})
	}
	return st
}

func totalsOnly(cur map[string]netCounter) NetStat {
	st := NetStat{Interfaces: []NetIface{}}
	names := make([]string, 0, len(cur))
	for name := range cur {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c := cur[name]
		st.TotalSentBytes += c.sent
		st.TotalRecvBytes += c.recv
		st.Interfaces = append(st.Interfaces, NetIface{Name: name, BytesSent: c.sent, BytesRecv: c.recv})
	}
	return st
}

func topProcs(n int) []TopProc {
	procs, err := process.Processes()
	if err != nil || len(procs) == 0 {
		return []TopProc{}
	}
	items := make([]TopProc, 0, len(procs))
	for _, p := range procs {
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}
		cpuPct, _ := p.CPUPercent()
		user, _ := p.Username()
		items = append(items, TopProc{PID: p.Pid, User: user, Name: name, CPUPercent: round2(cpuPct)})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CPUPercent > items[j].CPUPercent })
	if len(items) > n {
		items = items[:n]
	}
	return items
}

func rate(cur, prev uint64, sec float64) float64 {
	if cur < prev || sec <= 0 {
		return 0
	}
	return float64(cur-prev) / sec
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
