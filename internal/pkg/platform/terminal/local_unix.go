//go:build unix

package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/creack/pty"
)

type localSession struct {
	ptmx *os.File
	cmd  *exec.Cmd
}

/**
 * openLocal 在 Unix 上用 creack/pty 挂默认 shell。
 */
func openLocal(cols, rows uint16) (Session, error) {
	// 1. 选择 shell：环境变量优先，否则 bash/sh
	shell := strings.TrimSpace(os.Getenv("SHELL"))
	if shell == "" {
		if _, err := os.Stat("/bin/bash"); err == nil {
			shell = "/bin/bash"
		} else {
			shell = "/bin/sh"
		}
	}
	cmd := exec.Command(shell)
	cmd.Env = os.Environ()

	// 2. 启动 PTY 并设置初始窗口
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, fmt.Errorf("启动本机终端失败: %w", err)
	}
	return &localSession{ptmx: ptmx, cmd: cmd}, nil
}

func (s *localSession) Read(p []byte) (int, error)  { return s.ptmx.Read(p) }
func (s *localSession) Write(p []byte) (int, error) { return s.ptmx.Write(p) }

func (s *localSession) Resize(cols, rows int) error {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	return pty.Setsize(s.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

func (s *localSession) Close() error {
	_ = s.ptmx.Close()
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_, _ = s.cmd.Process.Wait()
	}
	return nil
}
