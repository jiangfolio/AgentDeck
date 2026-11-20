package process

import (
	"fmt"
	"sort"
	"strings"

	platproc "github.com/jiangbyte/agentdeck/internal/pkg/platform/process"
	"github.com/shirou/gopsutil/v4/process"
)

// Service 进程管理。
type Service struct{}

func NewService() *Service {
	return &Service{}
}

/**
 * List 枚举进程后按条件过滤，全量返回（不分页）。
 */
func (s *Service) List(q ListQuery) (*ListResult, error) {
	// 1. 枚举系统进程并组装精简字段
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("枚举进程失败: %w", err)
	}
	keyword := strings.ToLower(strings.TrimSpace(q.Keyword))
	userFilter := strings.ToLower(strings.TrimSpace(q.User))
	items := make([]Item, 0, len(procs))
	for _, p := range procs {
		item, ok := buildItem(p)
		if !ok {
			continue
		}
		// 2. keyword 匹配 name/cmdline；user 精确忽略大小写
		if keyword != "" {
			if !strings.Contains(strings.ToLower(item.Name), keyword) &&
				!strings.Contains(strings.ToLower(item.Cmdline), keyword) {
				continue
			}
		}
		if userFilter != "" && strings.ToLower(item.Username) != userFilter {
			continue
		}
		items = append(items, item)
	}

	// 3. 按 PID 排序后全量返回
	sort.Slice(items, func(i, j int) bool { return items[i].PID < items[j].PID })
	return &ListResult{Items: items, Total: len(items)}, nil
}

/**
 * Detail 返回单个进程详情。
 */
func (s *Service) Detail(pid int32) (*Detail, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("进程不存在")
	}
	item, ok := buildItem(p)
	if !ok {
		return nil, fmt.Errorf("读取进程失败")
	}
	d := &Detail{
		Item:          item,
		OpenFiles:     []string{},
		Envs:          []string{},
		Connections:   []Conn{},
		PartialErrors: map[string]string{},
		MemoryPercent: item.MemPercent,
	}
	// 1. 基础字段：失败只记到 partial_errors
	if ct, err := p.CreateTime(); err == nil {
		d.CreateTime = ct
	} else {
		d.PartialErrors["create_time"] = "无权限或进程已退出"
	}
	if n, err := p.NumThreads(); err == nil {
		d.NumThreads = n
	}
	if n, err := p.NumFDs(); err == nil {
		d.NumFDs = n
	}
	if mi, err := p.MemoryInfo(); err == nil && mi != nil {
		d.MemoryRSS = mi.RSS
		d.MemoryVMS = mi.VMS
	} else {
		d.PartialErrors["memory"] = "读取内存失败"
	}
	if io, err := p.IOCounters(); err == nil && io != nil {
		d.IOReadBytes = io.ReadBytes
		d.IOWriteBytes = io.WriteBytes
		if d.IOReadBytes == 0 {
			d.IOReadBytes = io.DiskReadBytes
		}
		if d.IOWriteBytes == 0 {
			d.IOWriteBytes = io.DiskWriteBytes
		}
	}
	if cwd, err := p.Cwd(); err == nil {
		d.Cwd = cwd
	}
	if exe, err := p.Exe(); err == nil {
		d.Exe = exe
	}
	if args, err := p.CmdlineSlice(); err == nil {
		d.CmdlineArgs = args
	}
	// 2. 打开文件、环境变量、连接各自截断，失败互不影响
	if files, err := p.OpenFiles(); err == nil {
		for _, f := range files {
			if f.Path == "" {
				continue
			}
			d.OpenFiles = append(d.OpenFiles, f.Path)
			if len(d.OpenFiles) >= 200 {
				break
			}
		}
	} else {
		d.PartialErrors["open_files"] = "无权限或不支持"
	}
	if envs, err := p.Environ(); err == nil {
		if len(envs) > 200 {
			envs = envs[:200]
		}
		d.Envs = envs
	} else {
		d.PartialErrors["envs"] = "无权限或不支持"
	}
	if conns, err := p.ConnectionsMax(200); err == nil {
		for _, c := range conns {
			d.Connections = append(d.Connections, Conn{
				Local:  addrText(c.Laddr.IP, c.Laddr.Port),
				Remote: addrText(c.Raddr.IP, c.Raddr.Port),
				Status: c.Status,
				Type:   connType(c.Type),
			})
		}
	} else {
		d.PartialErrors["connections"] = "无权限或不支持"
	}
	return d, nil
}

func addrText(ip string, port uint32) string {
	if ip == "" && port == 0 {
		return ""
	}
	if port == 0 {
		return ip
	}
	return fmt.Sprintf("%s:%d", ip, port)
}

func connType(t uint32) string {
	switch t {
	case 1:
		return "tcp"
	case 2:
		return "udp"
	default:
		return fmt.Sprintf("%d", t)
	}
}

/**
 * Kill 结束进程或进程树。
 */
func (s *Service) Kill(req KillRequest) error {
	return platproc.Kill(platproc.KillOptions{
		PID:         req.PID,
		Tree:        req.Tree,
		Force:       req.Force,
		ConfirmSelf: req.ConfirmSelf,
	})
}

func buildItem(p *process.Process) (Item, bool) {
	name, _ := p.Name()
	if name == "" {
		return Item{}, false
	}
	ppid, _ := p.Ppid()
	user, _ := p.Username()
	cpu, _ := p.CPUPercent()
	mem, _ := p.MemoryPercent()
	statusArr, _ := p.Status()
	status := ""
	if len(statusArr) > 0 {
		status = statusArr[0]
	}
	cmdline, _ := p.Cmdline()
	if len(cmdline) > 512 {
		cmdline = cmdline[:512]
	}
	return Item{
		PID:        p.Pid,
		PPID:       ppid,
		Name:       name,
		Username:   user,
		CPUPercent: round2(cpu),
		MemPercent: mem,
		Status:     status,
		Cmdline:    cmdline,
	}, true
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
