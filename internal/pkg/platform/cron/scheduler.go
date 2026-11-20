package cron

import "fmt"

// JobSpec 同步到系统调度器的任务描述。
type JobSpec struct {
	ID      string
	Name    string
	Expr    string
	Command string
	WorkDir string
	Enabled bool
}

// Scheduler 跨平台定时任务适配。
type Scheduler interface {
	Apply(job JobSpec) error
	Remove(jobID string) error
	Reconcile(jobs []JobSpec) error
}

// TaskName 生成 Windows 任务名 / 标记用 ID。
func TaskName(jobID string) string {
	return fmt.Sprintf("AgentDeck-Cron-%s", jobID)
}
