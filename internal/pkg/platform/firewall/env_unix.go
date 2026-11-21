//go:build unix

package firewall

import "strings"

// enrichPath 补上 /usr/sbin:/sbin，避免部署或 sudo 后 PATH 找不到 ufw/iptables/pfctl。
func enrichPath(env []string) []string {
	const prefix = "/usr/sbin:/sbin:"
	for i, item := range env {
		if !strings.HasPrefix(item, "PATH=") {
			continue
		}
		path := item[len("PATH="):]
		if strings.Contains(path, "/usr/sbin") || strings.Contains(path, "/sbin") {
			return env
		}
		env[i] = "PATH=" + prefix + path
		return env
	}
	return append(env, "PATH="+prefix+"/usr/bin:/bin")
}
