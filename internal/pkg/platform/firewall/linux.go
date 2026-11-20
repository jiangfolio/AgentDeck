//go:build linux

package firewall

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func New() Manager {
	return &ufwManager{}
}

type ufwManager struct{}

func (m *ufwManager) Status() Status {
	st := Status{Backend: "ufw"}
	fillInstallCaps(&st)
	if ufwBin() == "" {
		st.Message = missingUfwMessage(st)
		return st
	}
	st.Installed = true
	st.CanOnline = false
	st.CanOffline = false
	bin := ufwBin()
	out, err := run(bin, "status", "verbose")
	st.Enabled = strings.Contains(out, "Status: active")
	if ver, vErr := run(bin, "version"); vErr == nil {
		st.Version = firstLine(ver)
	}
	if err != nil && st.Message == "" {
		st.Message = "读取 ufw 状态失败，可能需要 root 权限"
	}
	return st
}

func (m *ufwManager) Rules() ([]Rule, error) {
	bin, err := needUfw()
	if err != nil {
		return []Rule{}, err
	}
	out, err := run(bin, "status", "numbered")
	rules := parseUfw(out)
	if err != nil && len(rules) == 0 {
		return []Rule{}, errors.New("读取 ufw 规则失败，可能需要 root 权限")
	}
	return rules, nil
}

func (m *ufwManager) SetEnabled(enabled bool) error {
	bin, err := needUfw()
	if err != nil {
		return err
	}
	var out string
	if enabled {
		out, err = run(bin, "--force", "enable")
	} else {
		out, err = run(bin, "disable")
	}
	if err != nil {
		return fmt.Errorf("%s", firstLine(out))
	}
	return nil
}

/**
 * Save 先按 id 删掉旧规则，再用 comment 前缀 agentdeck: 重新写入。
 */
func (m *ufwManager) Save(rule Rule) error {
	bin, err := needUfw()
	if err != nil {
		return err
	}
	// 1. 同 id 已存在则先删，避免更新变成重复规则
	rules, _ := m.Rules()
	for _, r := range rules {
		if r.ID == rule.ID {
			_ = m.Remove(rule.ID)
			break
		}
	}
	// 2. 组装 ufw allow/deny，来源、端口、协议按需附加
	verb := "allow"
	if rule.Action == "deny" {
		verb = "deny"
	}
	args := []string{verb}
	if rule.IP != "" && !strings.EqualFold(rule.IP, "any") {
		args = append(args, "from", rule.IP)
	}
	args = append(args, "to", "any")
	if rule.Port != "" {
		args = append(args, "port", rule.Port)
	}
	if rule.Protocol == "tcp" || rule.Protocol == "udp" {
		args = append(args, "proto", rule.Protocol)
	}
	comment := "agentdeck:" + rule.ID
	if rule.Description != "" {
		comment += " " + rule.Description
	}
	args = append(args, "comment", comment)
	out, err := run(bin, args...)
	if err != nil {
		return fmt.Errorf("%s", firstLine(out))
	}
	return nil
}

func (m *ufwManager) Remove(id string) error {
	rules, err := m.Rules()
	if err != nil {
		return err
	}
	num := ""
	for _, r := range rules {
		if r.ID == id {
			num = strconv.Itoa(r.Priority)
			break
		}
	}
	if num == "" {
		return errors.New("规则不存在")
	}
	bin, bErr := needUfw()
	if bErr != nil {
		return bErr
	}
	out, err := run(bin, "--force", "delete", num)
	if err != nil {
		return fmt.Errorf("%s", firstLine(out))
	}
	return nil
}

func (m *ufwManager) Reset() error {
	bin, err := needUfw()
	if err != nil {
		return err
	}
	out, err := run(bin, "--force", "reset")
	if err != nil {
		return fmt.Errorf("%s", firstLine(out))
	}
	return nil
}

func parseUfw(out string) []Rule {
	rules := make([]Rule, 0)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[") {
			continue
		}
		comment := ""
		if i := strings.Index(line, "#"); i >= 0 {
			comment = strings.TrimSpace(line[i+1:])
			line = strings.TrimSpace(line[:i])
		}
		j := strings.Index(line, "]")
		if j < 0 {
			continue
		}
		num := strings.TrimSpace(line[1:j])
		fields := strings.Fields(line[j+1:])
		if len(fields) < 4 || fields[2] != "IN" {
			continue
		}
		port, proto := splitPortProto(fields[0])
		action := "allow"
		if fields[1] == "DENY" || fields[1] == "REJECT" {
			action = "deny"
		}
		ip := fields[3]
		if strings.EqualFold(ip, "Anywhere") || ip == "0.0.0.0/0" {
			ip = ""
		}
		id := "n:" + num
		desc := comment
		managed := false
		if strings.HasPrefix(comment, "agentdeck:") {
			body := strings.TrimPrefix(comment, "agentdeck:")
			parts := strings.SplitN(body, " ", 2)
			id = parts[0]
			managed = id != ""
			if len(parts) == 2 {
				desc = parts[1]
			} else {
				desc = ""
			}
		}
		priority, _ := strconv.Atoi(num)
		rules = append(rules, Rule{
			ID: id, Action: action, Protocol: proto, IP: ip, Port: port,
			Description: desc, Priority: priority, Managed: managed,
		})
	}
	return rules
}

func splitPortProto(to string) (string, string) {
	if to == "" || strings.EqualFold(to, "Anywhere") {
		return "", "any"
	}
	proto := "any"
	port := to
	if i := strings.LastIndex(to, "/"); i > 0 {
		port = to[:i]
		proto = strings.ToLower(to[i+1:])
	}
	return port, proto
}
