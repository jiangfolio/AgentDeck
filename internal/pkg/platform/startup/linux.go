//go:build linux

package startup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	linuxSelfUnit   = "agentdeck.service"
	linuxSelfDesktop = "agentdeck.desktop"
)

type linuxManager struct{}

// New 返回 Linux 自启管理器。
func New() Manager {
	return &linuxManager{}
}

func (m *linuxManager) SelfEnabled() (bool, error) {
	if systemdUserAvailable() {
		out, err := exec.Command("systemctl", "--user", "is-enabled", linuxSelfUnit).CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) == "enabled" {
			return true, nil
		}
		return false, nil
	}
	path := filepath.Join(xdgAutostartDir(), linuxSelfDesktop)
	_, err := os.Stat(path)
	return err == nil, nil
}

/**
 * SetSelfEnabled 安装或卸载 AgentDeck 用户级自启（systemd user 优先，否则 XDG）。
 */
func (m *linuxManager) SetSelfEnabled(enabled bool) error {
	exe, err := ExecutablePath()
	if err != nil {
		return err
	}
	if systemdUserAvailable() {
		return m.setSelfSystemd(enabled, exe)
	}
	return m.setSelfDesktop(enabled, exe)
}

func (m *linuxManager) setSelfSystemd(enabled bool, exe string) error {
	unitDir := filepath.Join(homeDir(), ".config", "systemd", "user")
	unitPath := filepath.Join(unitDir, linuxSelfUnit)
	if !enabled {
		_ = exec.Command("systemctl", "--user", "disable", "--now", linuxSelfUnit).Run()
		_ = os.Remove(unitPath)
		_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
		return nil
	}
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(`[Unit]
Description=AgentDeck
After=default.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure

[Install]
WantedBy=default.target
`, exe)
	if err := os.WriteFile(unitPath, []byte(content), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	out, err := exec.Command("systemctl", "--user", "enable", linuxSelfUnit).CombinedOutput()
	if err != nil {
		return fmt.Errorf("enable 失败: %v (%s)", err, string(out))
	}
	return nil
}

func (m *linuxManager) setSelfDesktop(enabled bool, exe string) error {
	dir := xdgAutostartDir()
	path := filepath.Join(dir, linuxSelfDesktop)
	if !enabled {
		return os.RemoveAll(path)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=AgentDeck
Exec=%s
X-GNOME-Autostart-enabled=true
Hidden=false
`, exe)
	return os.WriteFile(path, []byte(content), 0o644)
}

/**
 * ListPlatformEntries 扫描 XDG autostart 与 AgentDeck 管理的 desktop。
 */
func (m *linuxManager) ListPlatformEntries() ([]PlatformEntry, error) {
	dir := xdgAutostartDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []PlatformEntry{}, nil
		}
		return nil, err
	}
	var list []PlatformEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".desktop") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		name, execCmd, hidden := parseDesktop(string(data))
		id := strings.TrimSuffix(e.Name(), ".desktop")
		managed := strings.HasPrefix(e.Name(), "agentdeck-") || e.Name() == linuxSelfDesktop
		list = append(list, PlatformEntry{
			ID:       id,
			Name:     name,
			Command:  execCmd,
			Enabled:  !hidden,
			Managed:  managed,
			Source:   "xdg-autostart",
			ReadOnly: !managed,
		})
	}
	return list, nil
}

/**
 * ApplyEntry 写入 XDG .desktop（仅 managed 前缀）。
 */
func (m *linuxManager) ApplyEntry(e EntrySpec) error {
	dir := xdgAutostartDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "agentdeck-"+e.ID+".desktop")
	hidden := "false"
	if !e.Enabled {
		hidden = "true"
	}
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
Path=%s
X-GNOME-Autostart-enabled=%t
Hidden=%s
X-AgentDeck-ID=%s
`, e.Name, e.Command, e.WorkDir, e.Enabled, hidden, e.ID)
	return os.WriteFile(path, []byte(content), 0o644)
}

func (m *linuxManager) RemoveEntry(id string) error {
	path := filepath.Join(xdgAutostartDir(), "agentdeck-"+id+".desktop")
	return os.RemoveAll(path)
}

func (m *linuxManager) SetEntryEnabled(id string, enabled bool) error {
	path := filepath.Join(xdgAutostartDir(), "agentdeck-"+id+".desktop")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(data)
	text = replaceDesktopBool(text, "Hidden", !enabled)
	text = replaceDesktopBool(text, "X-GNOME-Autostart-enabled", enabled)
	return os.WriteFile(path, []byte(text), 0o644)
}

func systemdUserAvailable() bool {
	if err := exec.Command("systemctl", "--user", "status").Run(); err != nil {
		// status 非 0 也可能可用，再试 is-system-running
		_, err2 := exec.LookPath("systemctl")
		return err2 == nil && os.Getenv("XDG_RUNTIME_DIR") != ""
	}
	return true
}

func xdgAutostartDir() string {
	return filepath.Join(homeDir(), ".config", "autostart")
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}

func parseDesktop(content string) (name, execCmd string, hidden bool) {
	hidden = false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Name="):
			name = strings.TrimPrefix(line, "Name=")
		case strings.HasPrefix(line, "Exec="):
			execCmd = strings.TrimPrefix(line, "Exec=")
		case line == "Hidden=true":
			hidden = true
		case line == "X-GNOME-Autostart-enabled=false":
			hidden = true
		}
	}
	if name == "" {
		name = execCmd
	}
	return
}

func replaceDesktopBool(content, key string, val bool) string {
	lines := strings.Split(content, "\n")
	found := false
	target := fmt.Sprintf("%s=%t", key, val)
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), key+"=") {
			lines[i] = target
			found = true
		}
	}
	if !found {
		lines = append(lines, target)
	}
	return strings.Join(lines, "\n")
}
