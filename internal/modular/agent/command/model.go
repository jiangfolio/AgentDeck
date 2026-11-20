package command

// RunRequest 试运行命令请求。
type RunRequest struct {
	Command string `json:"command" binding:"required,max=2048"`
	WorkDir string `json:"work_dir" binding:"max=512"`
}

// RunResult 试运行结果。
type RunResult struct {
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	DurationMs int64  `json:"duration_ms"`
}
