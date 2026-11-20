//go:build windows

package process

import (
	"fmt"
	"os/exec"
	"strconv"

	"github.com/shirou/gopsutil/v4/process"
)

/**
 * killOne 调用 taskkill 结束单个进程。
 */
func killOne(pid int32, force bool) error {
	args := []string{"/PID", strconv.Itoa(int(pid))}
	if force {
		args = append(args, "/F")
	}
	out, err := exec.Command("taskkill", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("taskkill 失败: %v (%s)", err, string(out))
	}
	return nil
}

/**
 * killTree 使用 taskkill /T 结束进程树。
 */
func killTree(root int32, force bool) error {
	args := []string{"/PID", strconv.Itoa(int(root)), "/T"}
	if force {
		args = append(args, "/F")
	}
	out, err := exec.Command("taskkill", args...).CombinedOutput()
	if err != nil {
		// 回退：手动按 PPID 递归
		_ = out
		return killTreeManual(root, force)
	}
	return nil
}

func killTreeManual(root int32, force bool) error {
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
	var order []int32
	var walk func(int32)
	walk = func(pid int32) {
		for _, c := range children[pid] {
			walk(c)
		}
		order = append(order, pid)
	}
	walk(root)
	var last error
	for _, pid := range order {
		if err := killOne(pid, force); err != nil {
			last = err
		}
	}
	return last
}
