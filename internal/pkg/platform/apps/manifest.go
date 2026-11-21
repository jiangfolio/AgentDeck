package apps

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Manifest 绿色包内可选的元数据。
type Manifest struct {
	ID      string   `json:"id"`
	Version string   `json:"version"`
	OS      string   `json:"os"`
	Arch    string   `json:"arch"`
	Bins    []string `json:"bins"`
}

/**
 * ReadManifest 读取解压目录根部的 manifest.json；不存在则返回 nil。
 */
func ReadManifest(installDir string) (*Manifest, error) {
	path := filepath.Join(installDir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest.json 无效: %w", err)
	}
	return &m, nil
}

/**
 * ResolveIdentity 合并用户选择与 manifest，得到最终 id/version。
 */
func ResolveIdentity(selectedID, selectedVersion string, m *Manifest) (id, version string, err error) {
	id = strings.TrimSpace(selectedID)
	version = strings.TrimSpace(selectedVersion)
	if m != nil {
		if id == "" {
			id = strings.TrimSpace(m.ID)
		} else if m.ID != "" && !strings.EqualFold(m.ID, id) {
			return "", "", fmt.Errorf("包内 id=%s 与所选应用 %s 不一致", m.ID, id)
		}
		if version == "" {
			version = strings.TrimSpace(m.Version)
		} else if m.Version != "" && m.Version != version {
			return "", "", fmt.Errorf("包内 version=%s 与所选版本 %s 不一致", m.Version, version)
		}
		if m.OS != "" && !strings.EqualFold(m.OS, runtime.GOOS) {
			return "", "", fmt.Errorf("包目标系统 %s 与当前 %s 不匹配", m.OS, runtime.GOOS)
		}
		if m.Arch != "" && !strings.EqualFold(m.Arch, runtime.GOARCH) {
			return "", "", fmt.Errorf("包目标架构 %s 与当前 %s 不匹配", m.Arch, runtime.GOARCH)
		}
	}
	if id == "" {
		return "", "", fmt.Errorf("请选择应用或在包内提供 manifest.id")
	}
	if version == "" {
		return "", "", fmt.Errorf("请填写版本或在包内提供 manifest.version")
	}
	return id, version, nil
}

/**
 * VerifyBins 确认 layout/bin 或 manifest.bins 在安装目录下存在。
 */
func VerifyBins(installDir string, layoutBin string, manifestBins []string) error {
	check := func(rel string) error {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			return nil
		}
		full, err := SafeJoinUnder(installDir, rel)
		if err != nil {
			return err
		}
		st, err := os.Stat(full)
		if err != nil || st.IsDir() {
			return fmt.Errorf("缺少可执行文件: %s", rel)
		}
		return nil
	}
	if err := check(layoutBin); err != nil {
		return err
	}
	for _, b := range manifestBins {
		if err := check(b); err != nil {
			return err
		}
	}
	return nil
}
