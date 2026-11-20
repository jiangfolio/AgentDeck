//go:build windows

package firewall

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func New() Manager {
	return &netshManager{}
}

type netshManager struct{}

func (m *netshManager) Status() Status {
	st := Status{
		Backend: "netsh", Version: "Windows Firewall", OS: "windows",
		Package: "Windows Firewall", OfflineExt: []string{},
	}
	if _, err := exec.LookPath("netsh"); err != nil {
		st.Message = "未找到 netsh。Windows 防火墙是系统组件，不能在面板里安装"
		return st
	}
	st.Installed = true
	out, err := run("netsh", "advfirewall", "show", "allprofiles")
	if err != nil {
		st.Message = "读取防火墙状态失败"
		return st
	}
	st.Enabled = strings.Contains(strings.ToLower(out), "state") && strings.Contains(strings.ToLower(out), "on")
	if strings.Contains(strings.ToLower(out), "状态") && strings.Contains(out, "启用") {
		st.Enabled = true
	}
	return st
}

func (m *netshManager) Rules() ([]Rule, error) {
	out, err := run("netsh", "advfirewall", "firewall", "show", "rule", "name=all")
	if err != nil {
		return []Rule{}, errors.New("读取防火墙规则失败")
	}
	return parseNetsh(out), nil
}

func (m *netshManager) SetEnabled(enabled bool) error {
	state := "off"
	if enabled {
		state = "on"
	}
	out, err := run("netsh", "advfirewall", "set", "allprofiles", "state", state)
	if err != nil {
		return fmt.Errorf("%s", firstLine(out))
	}
	return nil
}

/**
 * Save 用规则名 agentdeck:<id> 删除后重建，保证更新幂等。
 */
func (m *netshManager) Save(rule Rule) error {
	name := "agentdeck:" + rule.ID
	_, _ = run("netsh", "advfirewall", "firewall", "delete", "rule", "name="+name)
	action := "allow"
	if rule.Action == "deny" {
		action = "block"
	}
	args := []string{"advfirewall", "firewall", "add", "rule", "name=" + name, "dir=in", "action=" + action, "enable=yes"}
	if rule.Protocol == "tcp" || rule.Protocol == "udp" {
		args = append(args, "protocol="+strings.ToUpper(rule.Protocol))
	} else {
		args = append(args, "protocol=any")
	}
	if rule.Port != "" {
		args = append(args, "localport="+strings.ReplaceAll(rule.Port, ":", "-"))
	}
	ip := rule.IP
	if ip == "" {
		ip = "any"
	}
	args = append(args, "remoteip="+ip)
	if rule.Description != "" {
		args = append(args, "description="+rule.Description)
	}
	out, err := run("netsh", args...)
	if err != nil {
		return fmt.Errorf("%s", firstLine(out))
	}
	return nil
}

func (m *netshManager) Remove(id string) error {
	rules, err := m.Rules()
	if err != nil {
		return err
	}
	name := ""
	for _, r := range rules {
		if r.ID == id {
			if r.Managed {
				name = "agentdeck:" + r.ID
			} else {
				name = r.ID
			}
			break
		}
	}
	if name == "" {
		return errors.New("规则不存在")
	}
	out, err := run("netsh", "advfirewall", "firewall", "delete", "rule", "name="+name)
	if err != nil {
		return fmt.Errorf("%s", firstLine(out))
	}
	return nil
}

func (m *netshManager) InstallOnline() error {
	return errors.New("Windows 防火墙是系统组件，不能在线安装")
}

func (m *netshManager) InstallOffline(string) error {
	return errors.New("Windows 防火墙是系统组件，不能离线安装")
}

func (m *netshManager) Reset() error {
	out, err := run("netsh", "advfirewall", "reset")
	if err != nil {
		return fmt.Errorf("%s", firstLine(out))
	}
	return nil
}

func parseNetsh(out string) []Rule {
	chunks := strings.Split(out, "Rule Name:")
	rules := make([]Rule, 0)
	priority := 0
	for _, chunk := range chunks[1:] {
		lines := strings.Split(chunk, "\n")
		name := strings.TrimSpace(lines[0])
		if name == "" {
			continue
		}
		fields := map[string]string{}
		for _, ln := range lines[1:] {
			i := strings.Index(ln, ":")
			if i <= 0 {
				continue
			}
			fields[strings.TrimSpace(ln[:i])] = strings.TrimSpace(ln[i+1:])
		}
		dir := strings.ToLower(fields["Direction"])
		if dir != "" && dir != "in" && !strings.Contains(fields["Direction"], "入") {
			continue
		}
		action := "allow"
		act := strings.ToLower(fields["Action"])
		if strings.Contains(act, "block") || strings.Contains(fields["Action"], "阻止") {
			action = "deny"
		}
		proto := strings.ToLower(fields["Protocol"])
		if proto == "" || proto == "any" {
			proto = "any"
		}
		port := fields["LocalPort"]
		if strings.EqualFold(port, "any") || port == "任何" {
			port = ""
		}
		port = strings.ReplaceAll(port, "-", ":")
		ip := fields["RemoteIP"]
		if strings.EqualFold(ip, "any") || ip == "任何" {
			ip = ""
		}
		id := name
		managed := false
		desc := fields["Description"]
		if strings.HasPrefix(name, "agentdeck:") {
			id = strings.TrimPrefix(name, "agentdeck:")
			managed = true
		}
		priority++
		rules = append(rules, Rule{
			ID: id, Action: action, Protocol: proto, IP: ip, Port: port,
			Description: desc, Priority: priority, Managed: managed,
		})
	}
	return rules
}
