package startup

import (
	"fmt"
	"os"
	"path/filepath"
)

// EntrySpec AgentDeck 管理的用户级自启条目。
type EntrySpec struct {
	ID      string
	Name    string
	Command string
	WorkDir string
	Enabled bool
}

// PlatformEntry 扫描到的平台自启项。
type PlatformEntry struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Command  string `json:"command"`
	Enabled  bool   `json:"enabled"`
	Managed  bool   `json:"managed"`
	Source   string `json:"source"`
	ReadOnly bool   `json:"read_only"`
}

// Manager 跨平台开机自启适配。
type Manager interface {
	SelfEnabled() (bool, error)
	SetSelfEnabled(enabled bool) error
	ListPlatformEntries() ([]PlatformEntry, error)
	ApplyEntry(e EntrySpec) error
	RemoveEntry(id string) error
	SetEntryEnabled(id string, enabled bool) error
}

// ExecutablePath 返回当前可执行文件绝对路径。
func ExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取可执行路径失败: %w", err)
	}
	return filepath.Abs(exe)
}
