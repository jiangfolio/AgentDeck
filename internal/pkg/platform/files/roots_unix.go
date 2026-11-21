//go:build unix

package files

// Roots 返回 Unix 文件系统根入口。
func Roots() []string {
	return []string{"/"}
}
