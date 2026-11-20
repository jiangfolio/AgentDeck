//go:build windows

package startup

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jiangbyte/agentdeck/internal/config"
)

type windowsManager struct{}

// New 返回 Windows 自启管理器。
func New() Manager {
	return &windowsManager{}
}

func (m *windowsManager) SelfEnabled() (bool, error) {
	out, err := exec.Command("schtasks", "/Query", "/TN", config.StartupSelfTask).CombinedOutput()
	if err != nil {
		return false, nil
	}
	return strings.Contains(string(out), config.StartupSelfTask), nil
}

/**
 * SetSelfEnabled 用 schtasks 注册/删除登录自启。
 */
func (m *windowsManager) SetSelfEnabled(enabled bool) error {
	if !enabled {
		_ = exec.Command("schtasks", "/Delete", "/TN", config.StartupSelfTask, "/F").Run()
		return nil
	}
	exe, err := ExecutablePath()
	if err != nil {
		return err
	}
	args := []string{
		"/Create", "/TN", config.StartupSelfTask, "/SC", "ONLOGON",
		"/TR", fmt.Sprintf("\"%s\"", exe), "/F",
	}
	out, err := exec.Command("schtasks", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("创建自启任务失败: %v (%s)", err, string(out))
	}
	return nil
}

func (m *windowsManager) ListPlatformEntries() ([]PlatformEntry, error) {
	out, err := exec.Command("schtasks", "/Query", "/FO", "LIST").CombinedOutput()
	if err != nil {
		return []PlatformEntry{}, nil
	}
	var list []PlatformEntry
	var current string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "taskname:") {
			current = strings.TrimSpace(line[len("TaskName:"):])
			base := filepath.Base(strings.ReplaceAll(current, "\\", "/"))
			managed := strings.HasPrefix(base, config.StartupTaskPrefix) || base == config.StartupSelfTask
			list = append(list, PlatformEntry{
				ID:       base,
				Name:     base,
				Command:  "",
				Enabled:  true,
				Managed:  managed,
				Source:   "schtasks",
				ReadOnly: !managed,
			})
		}
	}
	return list, nil
}

func (m *windowsManager) ApplyEntry(e EntrySpec) error {
	name := config.StartupTaskPrefix + e.ID
	_ = m.RemoveEntry(e.ID)
	if !e.Enabled {
		return nil
	}
	cmd := e.Command
	if e.WorkDir != "" {
		cmd = fmt.Sprintf("cmd.exe /c cd /d \"%s\" && %s", e.WorkDir, e.Command)
	}
	args := []string{
		"/Create", "/TN", name, "/SC", "ONLOGON",
		"/TR", cmd, "/F",
	}
	out, err := exec.Command("schtasks", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("创建自启条目失败: %v (%s)", err, string(out))
	}
	return nil
}

func (m *windowsManager) RemoveEntry(id string) error {
	name := id
	if !strings.HasPrefix(name, config.StartupTaskPrefix) && name != config.StartupSelfTask {
		name = config.StartupTaskPrefix + id
	}
	_ = exec.Command("schtasks", "/Delete", "/TN", name, "/F").Run()
	return nil
}

func (m *windowsManager) SetEntryEnabled(id string, enabled bool) error {
	name := id
	if !strings.HasPrefix(name, config.StartupTaskPrefix) {
		name = config.StartupTaskPrefix + id
	}
	if enabled {
		out, err := exec.Command("schtasks", "/Change", "/TN", name, "/ENABLE").CombinedOutput()
		if err != nil {
			return fmt.Errorf("启用失败: %v (%s)", err, string(out))
		}
		return nil
	}
	out, err := exec.Command("schtasks", "/Change", "/TN", name, "/DISABLE").CombinedOutput()
	if err != nil {
		return fmt.Errorf("禁用失败: %v (%s)", err, string(out))
	}
	return nil
}
