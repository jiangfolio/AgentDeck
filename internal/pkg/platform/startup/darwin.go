//go:build darwin

package startup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const darwinSelfLabel = "com.agentdeck.self"

type darwinManager struct{}

// New 返回 macOS 自启管理器。
func New() Manager {
	return &darwinManager{}
}

func (m *darwinManager) SelfEnabled() (bool, error) {
	path := filepath.Join(launchAgentsDir(), darwinSelfLabel+".plist")
	_, err := os.Stat(path)
	return err == nil, nil
}

/**
 * SetSelfEnabled 安装/卸载 LaunchAgent。
 */
func (m *darwinManager) SetSelfEnabled(enabled bool) error {
	exe, err := ExecutablePath()
	if err != nil {
		return err
	}
	dir := launchAgentsDir()
	path := filepath.Join(dir, darwinSelfLabel+".plist")
	if !enabled {
		_ = exec.Command("launchctl", "unload", path).Run()
		return os.RemoveAll(path)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key>
  <array><string>%s</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><false/>
</dict>
</plist>
`, darwinSelfLabel, exe)
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return err
	}
	_ = exec.Command("launchctl", "unload", path).Run()
	out, err := exec.Command("launchctl", "load", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load 失败: %v (%s)", err, string(out))
	}
	return nil
}

func (m *darwinManager) ListPlatformEntries() ([]PlatformEntry, error) {
	dir := launchAgentsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []PlatformEntry{}, nil
		}
		return nil, err
	}
	var list []PlatformEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".plist") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".plist")
		managed := strings.HasPrefix(id, "com.agentdeck.")
		list = append(list, PlatformEntry{
			ID:       id,
			Name:     id,
			Command:  "",
			Enabled:  true,
			Managed:  managed,
			Source:   "launchagents",
			ReadOnly: !managed,
		})
	}
	return list, nil
}

func (m *darwinManager) ApplyEntry(e EntrySpec) error {
	dir := launchAgentsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	label := "com.agentdeck." + e.ID
	path := filepath.Join(dir, label+".plist")
	runAtLoad := "true"
	if !e.Enabled {
		runAtLoad = "false"
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key>
  <array><string>/bin/sh</string><string>-c</string><string>%s</string></array>
  <key>WorkingDirectory</key><string>%s</string>
  <key>RunAtLoad</key><%s/>
</dict>
</plist>
`, label, xmlEscape(e.Command), xmlEscape(e.WorkDir), runAtLoad)
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return err
	}
	_ = exec.Command("launchctl", "unload", path).Run()
	if e.Enabled {
		_ = exec.Command("launchctl", "load", path).Run()
	}
	return nil
}

func (m *darwinManager) RemoveEntry(id string) error {
	label := id
	if !strings.HasPrefix(label, "com.agentdeck.") {
		label = "com.agentdeck." + id
	}
	path := filepath.Join(launchAgentsDir(), label+".plist")
	_ = exec.Command("launchctl", "unload", path).Run()
	return os.RemoveAll(path)
}

func (m *darwinManager) SetEntryEnabled(id string, enabled bool) error {
	label := id
	if !strings.HasPrefix(label, "com.agentdeck.") {
		label = "com.agentdeck." + id
	}
	path := filepath.Join(launchAgentsDir(), label+".plist")
	if enabled {
		out, err := exec.Command("launchctl", "load", path).CombinedOutput()
		if err != nil {
			return fmt.Errorf("启用失败: %v (%s)", err, string(out))
		}
		return nil
	}
	_ = exec.Command("launchctl", "unload", path).Run()
	return nil
}

func launchAgentsDir() string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "Library", "LaunchAgents")
}

func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
