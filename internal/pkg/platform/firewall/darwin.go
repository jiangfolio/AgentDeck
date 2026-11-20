//go:build darwin

package firewall

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	anchorFile = "/etc/pf.anchors/agentdeck"
	metaFile   = "/etc/pf.anchors/agentdeck.json"
)

func New() Manager {
	return &pfManager{}
}

type pfManager struct{}

func (m *pfManager) Status() Status {
	st := Status{
		Backend: "pf", Version: "pfctl anchor agentdeck", OS: "darwin",
		Package: "pf", OfflineExt: []string{}, IsRoot: os.Geteuid() == 0,
	}
	if _, err := exec.LookPath("pfctl"); err != nil {
		st.Message = "未找到 pfctl。pf 是系统自带组件，不能在面板里安装"
		return st
	}
	st.Installed = true
	out, err := run("pfctl", "-s", "info")
	if err != nil {
		st.Message = "读取 pf 失败，可能需要 root 权限。规则仅写入 anchor agentdeck，不会修改 /etc/pf.conf"
		return st
	}
	low := strings.ToLower(out)
	st.Enabled = strings.Contains(low, "status: enabled") || strings.Contains(low, "status: enabled")
	st.Message = "规则载入 anchor agentdeck；若 pf.conf 未引用该 anchor，规则不会参与过滤"
	return st
}

func (m *pfManager) Rules() ([]Rule, error) {
	raw, err := os.ReadFile(metaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Rule{}, nil
		}
		return []Rule{}, errors.New("读取防火墙规则失败")
	}
	var rules []Rule
	if err := json.Unmarshal(raw, &rules); err != nil {
		return []Rule{}, errors.New("防火墙规则文件损坏")
	}
	if rules == nil {
		rules = []Rule{}
	}
	return rules, nil
}

func (m *pfManager) SetEnabled(enabled bool) error {
	arg := "-d"
	if enabled {
		arg = "-e"
	}
	out, err := run("pfctl", arg)
	if err != nil && !strings.Contains(strings.ToLower(out), "already") {
		return fmt.Errorf("%s", firstLine(out))
	}
	return nil
}

/**
 * Save 把规则写入 anchor 元数据并重新加载 agentdeck anchor。
 */
func (m *pfManager) Save(rule Rule) error {
	rules, err := m.Rules()
	if err != nil && !os.IsNotExist(err) && len(rules) == 0 {
		rules = []Rule{}
	}
	// 1. 同 id 替换，否则追加
	rule.Managed = true
	replaced := false
	for i := range rules {
		if rules[i].ID == rule.ID {
			rule.Priority = rules[i].Priority
			rules[i] = rule
			replaced = true
			break
		}
	}
	if !replaced {
		rule.Priority = len(rules) + 1
		rules = append(rules, rule)
	}
	// 2. 落盘并加载 anchor，不改 /etc/pf.conf
	return m.write(rules)
}

func (m *pfManager) Remove(id string) error {
	rules, err := m.Rules()
	if err != nil {
		return err
	}
	next := make([]Rule, 0, len(rules))
	found := false
	for _, r := range rules {
		if r.ID == id {
			found = true
			continue
		}
		next = append(next, r)
	}
	if !found {
		return errors.New("规则不存在")
	}
	return m.write(next)
}

func (m *pfManager) InstallOnline() error {
	return errors.New("pf 是系统自带组件，不能在线安装")
}

func (m *pfManager) InstallOffline(string) error {
	return errors.New("pf 是系统自带组件，不能离线安装")
}

func (m *pfManager) Reset() error {
	return m.write([]Rule{})
}

func (m *pfManager) write(rules []Rule) error {
	for i := range rules {
		rules[i].Managed = true
		rules[i].Priority = i + 1
	}
	raw, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return errors.New("生成规则失败")
	}
	if err := os.WriteFile(metaFile, raw, 0o644); err != nil {
		return errors.New("写入防火墙规则失败，可能需要 root 权限")
	}
	var b strings.Builder
	for _, r := range rules {
		fmt.Fprintln(&b, pfLine(r))
	}
	if err := os.WriteFile(anchorFile, []byte(b.String()), 0o644); err != nil {
		return errors.New("写入 pf anchor 失败，可能需要 root 权限")
	}
	out, err := run("pfctl", "-a", "agentdeck", "-f", anchorFile)
	if err != nil {
		return fmt.Errorf("%s", firstLine(out))
	}
	return nil
}

func pfLine(r Rule) string {
	verb := "pass"
	if r.Action == "deny" {
		verb = "block"
	}
	from := "any"
	if r.IP != "" && !strings.EqualFold(r.IP, "any") {
		from = r.IP
	}
	proto := ""
	if r.Protocol == "tcp" || r.Protocol == "udp" {
		proto = " proto " + r.Protocol
	}
	port := ""
	if r.Port != "" {
		port = " port " + r.Port
	}
	return fmt.Sprintf("%s in quick%s from %s to any%s", verb, proto, from, port)
}
