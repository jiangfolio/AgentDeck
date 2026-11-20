//go:build !windows

package process

import (
	"fmt"
	"os"
	"syscall"

	"github.com/shirou/gopsutil/v4/process"
)

/**
 * killOne 向目标进程发送 TERM 或 KILL。
 */
func killOne(pid int32, force bool) error {
	p, err := os.FindProcess(int(pid))
	if err != nil {
		return fmt.Errorf("进程不存在: %w", err)
	}
	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}
	if err := p.Signal(sig); err != nil {
		return fmt.Errorf("发送信号失败: %w", err)
	}
	return nil
}

/**
 * killTree 先收集子进程再由叶到根结束，避免孤儿残留。
 */
func killTree(root int32, force bool) error {
	// 1. 枚举全部进程，按 PPID 建树
	procs, err := process.Processes()
	if err != nil {
		return fmt.Errorf("枚举进程失败: %w", err)
	}
	children := map[int32][]int32{}
	for _, p := range procs {
		ppid, err := p.Ppid()
		if err != nil {
			continue
		}
		children[ppid] = append(children[ppid], p.Pid)
	}

	// 2. DFS 收集 root 子树（含自身）
	var order []int32
	var walk func(int32)
	walk = func(pid int32) {
		for _, c := range children[pid] {
			walk(c)
		}
		order = append(order, pid)
	}
	walk(root)

	// 3. 先子后父依次结束；忽略已退出进程错误
	var last error
	for _, pid := range order {
		if err := killOne(pid, force); err != nil {
			last = err
		}
	}
	return last
}
