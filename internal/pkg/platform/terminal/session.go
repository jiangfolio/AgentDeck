package terminal

import "io"

// Session 交互式终端后端（本机 PTY 或 SSH）。
type Session interface {
	io.ReadWriteCloser
	Resize(cols, rows int) error
}

// OpenLocal 打开本机交互 shell。
func OpenLocal(cols, rows uint16) (Session, error) {
	return openLocal(cols, rows)
}
