package firewall

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

func run(name string, args ...string) (string, error) {
	return runCtx(2*time.Minute, nil, name, args...)
}

// runCtx 带超时执行命令，可选附加环境变量。超时后返回明确错误，避免安装一直挂住。
func runCtx(timeout time.Duration, env []string, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = mergeEnv(env)
	out, err := cmd.CombinedOutput()
	text := string(out)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return text, errors.New("安装超时，请稍后在系统里确认是否已装上")
	}
	return text, err
}

// mergeEnv 合并额外环境变量；各平台可再补 PATH（见 env_*.go）。
func mergeEnv(extra []string) []string {
	env := append([]string{}, os.Environ()...)
	if len(extra) > 0 {
		env = append(env, extra...)
	}
	return enrichPath(env)
}

// usefulLine 取命令输出里最后一条有信息量的行，跳过 apt 的进度提示。
func usefulLine(s string) string {
	picked := ""
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		low := strings.ToLower(line)
		if strings.Contains(low, "reading package") ||
			strings.Contains(low, "building dependency") ||
			strings.Contains(low, "reading state") {
			continue
		}
		picked = line
	}
	if picked == "" {
		return firstLine(s)
	}
	if len(picked) > 180 {
		picked = picked[:180]
	}
	return picked
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 180 {
		s = s[:180]
	}
	if s == "" {
		return "命令执行失败"
	}
	return s
}
