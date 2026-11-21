package apps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	platproc "github.com/jiangbyte/agentdeck/internal/pkg/platform/process"
	"github.com/shirou/gopsutil/v4/process"
)

// Layout 目录中定义的二进制与启动参数（不做配置文件管理）。
type Layout struct {
	Bin      string            `json:"bin"`
	Env      map[string]string `json:"env,omitempty"`
	Start    []string          `json:"start,omitempty"`
	Runnable *bool             `json:"runnable,omitempty"`
}

// IsRunnable 默认可运行；显式 false 时不可启停。
func (l Layout) IsRunnable() bool {
	if l.Runnable == nil {
		return len(l.Start) > 0
	}
	return *l.Runnable
}

/**
 * PrepareCtx 为指定应用填充路径上下文；工作目录即应用目录。
 */
func PrepareCtx(appID string, layout Layout) (ExpandCtx, error) {
	installDir := AppDir(appID)
	binRel := Expand(layout.Bin, ExpandCtx{InstallDir: installDir})
	binAbs := ""
	var err error
	if binRel != "" {
		binAbs, err = SafeJoinUnder(installDir, binRel)
		if err != nil {
			return ExpandCtx{}, err
		}
	}
	return ExpandCtx{
		InstallDir: installDir,
		Workdir:    installDir,
		Bin:        binAbs,
	}, nil
}

/**
 * Start 按 layout.start 拉起进程，返回子进程 pid（由调用方登记到库）。
 */
func Start(appID string, layout Layout) (int, error) {
	// 1. 校验可运行并准备路径
	if !layout.IsRunnable() {
		return 0, fmt.Errorf("该应用未配置启停，请在目录中自行管理")
	}
	ctx, err := PrepareCtx(appID, layout)
	if err != nil {
		return 0, err
	}
	if _, err := os.Stat(ctx.InstallDir); err != nil {
		return 0, fmt.Errorf("应用目录不存在")
	}
	if len(layout.Start) == 0 {
		return 0, fmt.Errorf("未配置 start 命令")
	}
	// 2. 展开命令行；日志写到旁路目录
	args := make([]string, 0, len(layout.Start))
	for _, p := range layout.Start {
		args = append(args, Expand(p, ctx))
	}
	bin := args[0]
	cmdArgs := args[1:]
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Dir = ctx.Workdir
	cmd.Env = os.Environ()
	for k, v := range layout.Env {
		cmd.Env = append(cmd.Env, k+"="+Expand(v, ctx))
	}
	logDir := ServiceDir(appID)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return 0, err
	}
	logPath := filepath.Join(logDir, "start.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	// 3. 启动并返回 pid
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("启动失败: %w", err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	return pid, nil
}

/**
 * Stop 按登记的 pid 结束进程。
 */
func Stop(pid int) error {
	if pid <= 0 {
		return nil
	}
	if !PidAlive(pid) {
		return nil
	}
	_ = platproc.Kill(platproc.KillOptions{PID: int32(pid), Force: false})
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !PidAlive(pid) {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	_ = platproc.Kill(platproc.KillOptions{PID: int32(pid), Force: true, Tree: true})
	return nil
}

// PidAlive 判断进程是否仍存在。
func PidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	ok, err := process.PidExists(int32(pid))
	return err == nil && ok
}

/**
 * StatusByPid 根据登记 pid 返回运行状态。
 */
func StatusByPid(pid int) (running bool, alivePid int) {
	if !PidAlive(pid) {
		return false, 0
	}
	return true, pid
}
