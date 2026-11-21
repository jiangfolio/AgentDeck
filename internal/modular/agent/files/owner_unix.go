//go:build unix

package files

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

func fileOwner(info os.FileInfo) string {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return ""
	}
	uid := strconv.FormatUint(uint64(st.Uid), 10)
	gid := strconv.FormatUint(uint64(st.Gid), 10)
	uname, gname := uid, gid
	if u, err := user.LookupId(uid); err == nil && u.Username != "" {
		uname = u.Username
	}
	if g, err := user.LookupGroupId(gid); err == nil && g.Name != "" {
		gname = g.Name
	}
	return fmt.Sprintf("%s (%s) / %s (%s)", uname, uid, gname, gid)
}
