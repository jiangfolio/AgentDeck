//go:build linux

package firewall

import (
	"errors"
	"strconv"
	"strings"
	"time"
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
	out, err := runUfw(bin, "status", "verbose")
	st.Enabled = ufwActive(out)
	if ver, vErr := run(bin, "version"); vErr == nil {
		st.Version = firstLine(ver)
	}
	if msg := ufwErrMessage(out, err); msg != "" {
		st.Message = "读取 ufw 状态失败：" + msg
	}
	return st
}

func (m *ufwManager) Rules() ([]Rule, error) {
	bin, err := needUfw()
	if err != nil {
		return []Rule{}, err
	}
	out, err := runUfw(bin, "status", "numbered")
	rules := parseUfw(out)
	if msg := ufwErrMessage(out, err); msg != "" && len(rules) == 0 {
		return []Rule{}, errors.New("读取 ufw 规则失败：" + msg)
	}
	return rules, nil
}

/**
 * SetEnabled 开关防火墙，成功后再读一次状态，避免命令“假成功”。
 */
func (m *ufwManager) SetEnabled(enabled bool) error {
	bin, err := needUfw()
	if err != nil {
		return err
	}
	// 1. 执行 enable/disable；输出里带 ERROR 也视为失败
	var out string
	if enabled {
		out, err = runUfw(bin, "--force", "enable")
	} else {
		out, err = runUfw(bin, "disable")
	}
	if msg := ufwErrMessage(out, err); msg != "" {
		return errors.New(msg)
	}
	// 2. 再读状态确认，防止页面显示已开启但实际仍是关
	st := m.Status()
	if st.Message != "" && strings.Contains(st.Message, "读取 ufw 状态失败") {
		return errors.New(st.Message)
	}
	if st.Enabled != enabled {
		if enabled {
			return errors.New("已执行开启，但 ufw 仍未处于启用状态")
		}
		return errors.New("已执行关闭，但 ufw 仍处于启用状态")
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
	out, err := runUfw(bin, args...)
	if msg := ufwErrMessage(out, err); msg != "" {
		return errors.New(msg)
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
	out, err := runUfw(bin, "--force", "delete", num)
	if msg := ufwErrMessage(out, err); msg != "" {
		return errors.New(msg)
	}
	return nil
}

func (m *ufwManager) Reset() error {
	bin, err := needUfw()
	if err != nil {
		return err
	}
	out, err := runUfw(bin, "--force", "reset")
	if msg := ufwErrMessage(out, err); msg != "" {
		return errors.New(msg)
	}
	return nil
}

// runUfw 优先直接跑；非 root 时走免密 sudo，并保证 PATH 含 sbin。
func runUfw(bin string, args ...string) (string, error) {
	return runRoot(2*time.Minute, nil, bin, args...)
}

func ufwActive(out string) bool {
	low := strings.ToLower(out)
	if strings.Contains(low, "status: inactive") || strings.Contains(out, "状态：不活动") || strings.Contains(out, "状态: 不活动") {
		return false
	}
	return strings.Contains(low, "status: active") ||
		strings.Contains(out, "状态：激活") ||
		strings.Contains(out, "状态: 激活") ||
		strings.Contains(out, "状态：启用") ||
		strings.Contains(out, "状态: 启用")
}

// ufwErrMessage 把命令失败收成可读原因；exit=0 但输出含 ERROR 也算失败。
func ufwErrMessage(out string, err error) string {
	line := usefulLine(out)
	low := strings.ToLower(line + "\n" + out)
	needRoot := strings.Contains(low, "need to be root") ||
		strings.Contains(out, "需要 root") ||
		strings.Contains(out, "必须以 root")
	hasErr := err != nil || strings.Contains(low, "error:") || needRoot
	if !hasErr {
		return ""
	}
	if needRoot {
		return "需要 root 权限，请用 make dev-api-sudo 启动后端"
	}
	if line != "" && line != "命令执行失败" {
		return stripPkgPath(line)
	}
	if err != nil {
		return err.Error()
	}
	return "命令执行失败"
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
