package process

import (
	"fmt"
	"os"
)

// KillOptions 结束进程参数。
type KillOptions struct {
	PID   int32
	Tree  bool
	Force bool
	// ConfirmSelf 结束 AgentDeck 自身时必须为 true
	ConfirmSelf bool
}

/**
 * Kill 按选项结束进程；禁止 PID≤1，结束自身需显式确认。
 */
func Kill(opts KillOptions) error {
	// 1. 校验 PID 与自身保护
	if opts.PID <= 1 {
		return fmt.Errorf("禁止结束系统关键进程")
	}
	self := int32(os.Getpid())
	if opts.PID == self && !opts.ConfirmSelf {
		return fmt.Errorf("结束 AgentDeck 自身需显式确认")
	}
	// 2. 按需递归结束子树，或仅结束目标进程
	if opts.Tree {
		return killTree(opts.PID, opts.Force)
	}
	return killOne(opts.PID, opts.Force)
}
