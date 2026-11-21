//go:build windows

package files

import (
	"os"
)

// Roots 扫描本机存在的盘符，如 C:\、D:\。
func Roots() []string {
	out := make([]string, 0, 8)
	for c := 'A'; c <= 'Z'; c++ {
		root := string(c) + `:\`
		if st, err := os.Stat(root); err == nil && st.IsDir() {
			out = append(out, root)
		}
	}
	return out
}
