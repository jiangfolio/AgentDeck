//go:build windows

package files

import "os"

func fileOwner(os.FileInfo) string {
	return ""
}
