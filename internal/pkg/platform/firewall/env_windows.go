//go:build windows

package firewall

// enrichPath Windows 走系统 PATH / netsh，不改 PATH。
func enrichPath(env []string) []string {
	return env
}
