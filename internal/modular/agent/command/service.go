package command

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const runTimeout = 15 * time.Second
const outputLimit = 8000

// Service 命令试运行。
type Service struct{}

func NewService() *Service {
	return &Service{}
}

/**
 * Run 在受限超时内执行一次命令，供表单试跑（如 echo Hello）。
 */
func (s *Service) Run(req RunRequest) (*RunResult, error) {
	cmdText := strings.TrimSpace(req.Command)
	if cmdText == "" {
		return nil, fmt.Errorf("命令不能为空")
	}

	// 1. 工作目录：未填则用当前用户主目录
	workDir := strings.TrimSpace(req.WorkDir)
	if workDir == "" {
		if h, err := os.UserHomeDir(); err == nil {
			workDir = h
		}
	}

	// 2. 按平台组装 shell，带超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", cmdText)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", cmdText)
	}
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 3. 执行并截断输出，避免试跑刷爆响应
	start := time.Now()
	err := cmd.Run()
	dur := time.Since(start).Milliseconds()

	result := &RunResult{
		ExitCode:   0,
		Stdout:     truncate(stdout.String(), outputLimit),
		Stderr:     truncate(stderr.String(), outputLimit),
		DurationMs: dur,
	}
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.ExitCode = -1
			if result.Stderr == "" {
				result.Stderr = "执行超时"
			}
			return result, nil
		}
		if ee, ok := err.(*exec.ExitError); ok {
			result.ExitCode = ee.ExitCode()
			return result, nil
		}
		return nil, fmt.Errorf("执行失败: %v", err)
	}
	return result, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
