//go:build windows

package terminal

import (
	"fmt"
	"sync"

	"github.com/UserExistsError/conpty"
)

type localSession struct {
	cpty *conpty.ConPty
	once sync.Once
}

/**
 * openLocal 在 Windows 上用 ConPTY 挂 powershell。
 */
func openLocal(cols, rows uint16) (Session, error) {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	cpty, err := conpty.Start(
		`powershell.exe -NoLogo`,
		conpty.ConPtyDimensions(int(cols), int(rows)),
	)
	if err != nil {
		return nil, fmt.Errorf("启动本机终端失败: %w", err)
	}
	return &localSession{cpty: cpty}, nil
}

func (s *localSession) Read(p []byte) (int, error)  { return s.cpty.Read(p) }
func (s *localSession) Write(p []byte) (int, error) { return s.cpty.Write(p) }

func (s *localSession) Resize(cols, rows int) error {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	return s.cpty.Resize(cols, rows)
}

func (s *localSession) Close() error {
	var err error
	s.once.Do(func() {
		err = s.cpty.Close()
	})
	return err
}
