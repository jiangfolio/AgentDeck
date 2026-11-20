//go:build !windows

package cron

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jiangbyte/agentdeck/internal/config"
)

type unixScheduler struct{}

// New 返回 Unix crontab 调度器。
func New() Scheduler {
	return &unixScheduler{}
}

/**
 * Apply 写入/更新用户 crontab 中带 AgentDeck 标记的任务块。
 */
func (s *unixScheduler) Apply(job JobSpec) error {
	// 1. 先移除旧块，保证幂等
	if err := s.Remove(job.ID); err != nil {
		return err
	}
	if !job.Enabled {
		return nil
	}
	// 2. 生成 wrapper 脚本，调度器只调用脚本以便记日志
	wrapper, err := writeUnixWrapper(job)
	if err != nil {
		return err
	}
	// 3. 读出现有 crontab，追加标记块后写回
	lines, err := readCrontab()
	if err != nil {
		return err
	}
	block := buildUnixBlock(job.ID, job.Expr, wrapper)
	lines = append(lines, block...)
	return writeCrontab(lines)
}

/**
 * Remove 删除指定 job 的标记块与 wrapper。
 */
func (s *unixScheduler) Remove(jobID string) error {
	lines, err := readCrontab()
	if err != nil {
		return err
	}
	filtered := filterUnixBlocks(lines, jobID)
	if err := writeCrontab(filtered); err != nil {
		return err
	}
	_ = os.Remove(unixWrapperPath(jobID))
	return nil
}

/**
 * Reconcile 以 DB 任务为准：先清全部 AgentDeck 块，再按启用项重写。
 */
func (s *unixScheduler) Reconcile(jobs []JobSpec) error {
	lines, err := readCrontab()
	if err != nil {
		return err
	}
	cleaned := filterUnixBlocks(lines, "")
	if err := writeCrontab(cleaned); err != nil {
		return err
	}
	for _, job := range jobs {
		if !job.Enabled {
			_ = os.Remove(unixWrapperPath(job.ID))
			continue
		}
		if err := s.Apply(job); err != nil {
			return err
		}
	}
	return nil
}

func unixWrapperPath(jobID string) string {
	return filepath.Join(config.CronWrapperDir(), jobID+".sh")
}

/**
 * writeUnixWrapper 生成带锁与日志的 shell wrapper。
 */
func writeUnixWrapper(job JobSpec) (string, error) {
	if err := os.MkdirAll(config.CronWrapperDir(), 0o755); err != nil {
		return "", fmt.Errorf("创建 wrapper 目录失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(config.CronLogDir(), job.ID), 0o755); err != nil {
		return "", fmt.Errorf("创建日志目录失败: %w", err)
	}
	path := unixWrapperPath(job.ID)
	logDir := filepath.Join(config.CronLogDir(), job.ID)
	workDir := job.WorkDir
	if workDir == "" {
		workDir = "$HOME"
	}
	script := fmt.Sprintf(`#!/bin/sh
set -e
LOCK="%s.lock"
LOGDIR="%s"
WORKDIR="%s"
CMD=%q
if [ -f "$LOCK" ]; then
  echo "{\"ts\":\"$(date -Iseconds)\",\"exit_code\":-1,\"duration_ms\":0,\"stdout\":\"\",\"stderr\":\"skipped: locked\"}" >> "$LOGDIR/runs.jsonl"
  exit 0
fi
trap 'rm -f "$LOCK"' EXIT
touch "$LOCK"
START=$(date +%%s%%N)
OUT=$(mktemp)
ERR=$(mktemp)
set +e
cd "$WORKDIR" 2>/dev/null || true
sh -c "$CMD" >"$OUT" 2>"$ERR"
CODE=$?
set -e
END=$(date +%%s%%N)
DUR=$(( (END - START) / 1000000 ))
OUT_TXT=$(head -c 4000 "$OUT" | sed 's/"/\\"/g' | tr '\n' ' ')
ERR_TXT=$(head -c 4000 "$ERR" | sed 's/"/\\"/g' | tr '\n' ' ')
rm -f "$OUT" "$ERR"
echo "{\"ts\":\"$(date -Iseconds)\",\"exit_code\":$CODE,\"duration_ms\":$DUR,\"stdout\":\"$OUT_TXT\",\"stderr\":\"$ERR_TXT\"}" >> "$LOGDIR/runs.jsonl"
exit $CODE
`, path, logDir, workDir, job.Command)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		return "", fmt.Errorf("写入 wrapper 失败: %w", err)
	}
	return path, nil
}

func buildUnixBlock(jobID, expr, wrapper string) []string {
	return []string{
		fmt.Sprintf("# AgentDeck Cron Start %s", jobID),
		fmt.Sprintf("%s %s", expr, wrapper),
		fmt.Sprintf("# AgentDeck Cron End %s", jobID),
	}
}

/**
 * filterUnixBlocks 去掉指定 job 或全部 AgentDeck 块。
 * jobID 为空表示清除所有 AgentDeck 标记块。
 */
func filterUnixBlocks(lines []string, jobID string) []string {
	out := make([]string, 0, len(lines))
	skipping := false
	targetStart := ""
	if jobID != "" {
		targetStart = "# AgentDeck Cron Start " + jobID
	}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "# AgentDeck Cron Start ") {
			if jobID == "" || trim == targetStart {
				skipping = true
				continue
			}
		}
		if skipping {
			if strings.HasPrefix(trim, "# AgentDeck Cron End ") {
				skipping = false
			}
			continue
		}
		out = append(out, line)
	}
	return out
}

func readCrontab() ([]string, error) {
	out, err := exec.Command("crontab", "-l").CombinedOutput()
	if err != nil {
		text := string(out)
		// 无 crontab 时多数系统返回非 0
		if strings.Contains(text, "no crontab") || len(strings.TrimSpace(text)) == 0 {
			return []string{}, nil
		}
		// 某些发行版 stderr 含 no crontab
		if strings.Contains(strings.ToLower(text), "no crontab") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("读取 crontab 失败: %v (%s)", err, text)
	}
	text := strings.ReplaceAll(string(out), "\r\n", "\n")
	if strings.TrimSpace(text) == "" {
		return []string{}, nil
	}
	return strings.Split(strings.TrimRight(text, "\n"), "\n"), nil
}

func writeCrontab(lines []string) error {
	content := strings.Join(lines, "\n")
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("写入 crontab 失败: %v (%s)", err, string(out))
	}
	return nil
}
