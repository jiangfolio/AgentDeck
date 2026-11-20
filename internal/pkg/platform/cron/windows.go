//go:build windows

package cron

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jiangbyte/agentdeck/internal/config"
)

type windowsScheduler struct{}

// New 返回 Windows schtasks 调度器。
func New() Scheduler {
	return &windowsScheduler{}
}

/**
 * Apply 通过 schtasks 注册/更新登录用户下的计划任务。
 */
func (s *windowsScheduler) Apply(job JobSpec) error {
	// 1. 先删旧任务，保证幂等
	_ = s.Remove(job.ID)
	if !job.Enabled {
		return nil
	}
	// 2. 写 PowerShell wrapper
	wrapper, err := writeWindowsWrapper(job)
	if err != nil {
		return err
	}
	// 3. 每分钟触发，由脚本执行命令并写日志
	name := TaskName(job.ID)
	tr := fmt.Sprintf(`powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%s"`, wrapper)
	args := []string{"/Create", "/TN", name, "/SC", "MINUTE", "/MO", "1", "/TR", tr, "/F"}
	out, err := exec.Command("schtasks", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("创建计划任务失败: %v (%s)", err, string(out))
	}
	return nil
}

/**
 * Remove 删除 schtasks 任务与 wrapper。
 */
func (s *windowsScheduler) Remove(jobID string) error {
	name := TaskName(jobID)
	_ = exec.Command("schtasks", "/Delete", "/TN", name, "/F").Run()
	_ = os.Remove(windowsWrapperPath(jobID))
	return nil
}

/**
 * Reconcile 删除名称前缀匹配的旧任务后按 DB 重建启用项。
 */
func (s *windowsScheduler) Reconcile(jobs []JobSpec) error {
	out, _ := exec.Command("schtasks", "/Query", "/FO", "LIST", "/V").CombinedOutput()
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "taskname:") {
			name := strings.TrimSpace(line[len("TaskName:"):])
			base := filepath.Base(strings.ReplaceAll(name, "\\", "/"))
			if strings.HasPrefix(base, config.CronTaskNamePrefix) {
				_ = exec.Command("schtasks", "/Delete", "/TN", name, "/F").Run()
			}
		}
	}
	for _, job := range jobs {
		if !job.Enabled {
			continue
		}
		if err := s.Apply(job); err != nil {
			return err
		}
	}
	return nil
}

func windowsWrapperPath(jobID string) string {
	return filepath.Join(config.CronWrapperDir(), jobID+".ps1")
}

/**
 * writeWindowsWrapper 生成带锁与 JSONL 日志的 PS 脚本。
 */
func writeWindowsWrapper(job JobSpec) (string, error) {
	if err := os.MkdirAll(config.CronWrapperDir(), 0o755); err != nil {
		return "", err
	}
	logDir := filepath.Join(config.CronLogDir(), job.ID)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", err
	}
	path := windowsWrapperPath(job.ID)
	workDir := job.WorkDir
	if workDir == "" {
		workDir = os.Getenv("USERPROFILE")
	}

	var b strings.Builder
	b.WriteString("$ErrorActionPreference = 'Continue'\n")
	b.WriteString(fmt.Sprintf("$lock = '%s.lock'\n", escapePSSingle(path)))
	b.WriteString(fmt.Sprintf("$logDir = '%s'\n", escapePSSingle(logDir)))
	b.WriteString(fmt.Sprintf("$workDir = '%s'\n", escapePSSingle(workDir)))
	b.WriteString(fmt.Sprintf("$cmd = '%s'\n", escapePSSingle(job.Command)))
	b.WriteString(`if (Test-Path $lock) {
  $skip = '{"ts":"' + (Get-Date).ToString('o') + '","exit_code":-1,"duration_ms":0,"stdout":"","stderr":"skipped: locked"}'
  Add-Content -Path (Join-Path $logDir 'runs.jsonl') -Value $skip
  exit 0
}
New-Item -ItemType File -Path $lock -Force | Out-Null
try {
  $sw = [Diagnostics.Stopwatch]::StartNew()
  Set-Location -LiteralPath $workDir -ErrorAction SilentlyContinue
  $outFile = Join-Path $env:TEMP 'agentdeck-cron-out.txt'
  $errFile = Join-Path $env:TEMP 'agentdeck-cron-err.txt'
  $p = Start-Process -FilePath 'cmd.exe' -ArgumentList '/c', $cmd -NoNewWindow -Wait -PassThru -RedirectStandardOutput $outFile -RedirectStandardError $errFile
  $sw.Stop()
  $outTxt = ''
  $errTxt = ''
  if (Test-Path $outFile) { $outTxt = ((Get-Content $outFile -Raw) -replace '"','\"' -replace "[\r\n]+",' ') }
  if (Test-Path $errFile) { $errTxt = ((Get-Content $errFile -Raw) -replace '"','\"' -replace "[\r\n]+",' ') }
  if ($outTxt.Length -gt 4000) { $outTxt = $outTxt.Substring(0, 4000) }
  if ($errTxt.Length -gt 4000) { $errTxt = $errTxt.Substring(0, 4000) }
  $line = '{"ts":"' + (Get-Date).ToString('o') + '","exit_code":' + $p.ExitCode + ',"duration_ms":' + $sw.ElapsedMilliseconds + ',"stdout":"' + $outTxt + '","stderr":"' + $errTxt + '"}'
  Add-Content -Path (Join-Path $logDir 'runs.jsonl') -Value $line
  exit $p.ExitCode
} finally {
  Remove-Item -LiteralPath $lock -Force -ErrorAction SilentlyContinue
}
`)

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func escapePSSingle(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
