//go:build linux

package firewall

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func fillInstallCaps(st *Status) {
	st.OS = "linux"
	st.Package = "ufw"
	st.IsRoot = os.Geteuid() == 0
	st.Manager, st.OfflineExt = detectInstall()
	if st.OfflineExt == nil {
		st.OfflineExt = []string{}
	}
	st.CanOnline = st.Manager != ""
	st.CanOffline = len(st.OfflineExt) > 0
}

func missingUfwMessage(st Status) string {
	var b strings.Builder
	b.WriteString("未安装 ufw")
	switch {
	case st.CanOnline && st.CanOffline:
		fmt.Fprintf(&b, "。可使用 %s 在线安装，或上传 %s 离线安装", st.Manager, strings.Join(st.OfflineExt, "、"))
	case st.CanOnline:
		fmt.Fprintf(&b, "。可使用 %s 在线安装", st.Manager)
	case st.CanOffline:
		fmt.Fprintf(&b, "。可上传 %s 离线安装", strings.Join(st.OfflineExt, "、"))
	default:
		b.WriteString("。未找到包管理器，无法在面板内安装")
	}
	if !st.IsRoot {
		b.WriteString("。当前进程不是 root，将尝试免密 sudo")
	}
	return b.String()
}

// detectInstall 识别本机包管理器，并给出可接受的离线包后缀。
func detectInstall() (string, []string) {
	manager := ""
	switch {
	case hasBin("apt-get"):
		manager = "apt"
	case hasBin("dnf"):
		manager = "dnf"
	case hasBin("yum"):
		manager = "yum"
	case hasBin("pacman"):
		manager = "pacman"
	case hasBin("zypper"):
		manager = "zypper"
	case hasBin("apk"):
		manager = "apk"
	}
	exts := make([]string, 0, 2)
	if hasBin("dpkg") {
		exts = append(exts, ".deb")
	}
	if hasBin("rpm") {
		exts = append(exts, ".rpm")
	}
	return manager, exts
}

func hasBin(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// ufwBin 查找 ufw。发行版通常装在 /usr/sbin，普通用户的 PATH 里不一定有。
func ufwBin() string {
	if p, err := exec.LookPath("ufw"); err == nil {
		return p
	}
	for _, p := range []string{"/usr/sbin/ufw", "/sbin/ufw", "/usr/bin/ufw"} {
		fi, err := os.Stat(p)
		if err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return p
		}
	}
	return ""
}

func needUfw() (string, error) {
	bin := ufwBin()
	if bin == "" {
		return "", errors.New("未安装 ufw")
	}
	return bin, nil
}

/**
 * InstallOnline 用本机包管理器安装固定包名 ufw。
 */
func (m *ufwManager) InstallOnline() error {
	// 1. 已经能找到 ufw 就结束，避免重复安装
	if ufwBin() != "" {
		return nil
	}
	// 2. 只走已识别的包管理器，包名固定，不接受外部传入
	kind, _ := detectInstall()
	if kind == "" {
		return errors.New("未找到可用的包管理器")
	}
	// 3. 按包管理器安装；apt 先更新索引，更新失败仍尝试安装已有缓存
	env := []string{"DEBIAN_FRONTEND=noninteractive"}
	var out string
	var err error
	switch kind {
	case "apt":
		_, _ = runRoot(90*time.Second, env, "apt-get", "update")
		out, err = runRoot(3*time.Minute, env, "apt-get", "install", "-y", "-o", "Dpkg::Use-Pty=0", "ufw")
	case "dnf":
		out, err = runRoot(3*time.Minute, nil, "dnf", "install", "-y", "ufw")
	case "yum":
		out, err = runRoot(3*time.Minute, nil, "yum", "install", "-y", "ufw")
	case "pacman":
		out, err = runRoot(3*time.Minute, nil, "pacman", "-Sy", "--noconfirm", "ufw")
	case "zypper":
		out, err = runRoot(3*time.Minute, nil, "zypper", "--non-interactive", "install", "ufw")
	case "apk":
		out, err = runRoot(3*time.Minute, nil, "apk", "add", "--no-cache", "ufw")
	default:
		return errors.New("未找到可用的包管理器")
	}
	if err != nil {
		log.Printf("防火墙在线安装失败 manager=%s err=%s", kind, usefulLine(out))
		return publicInstallErr(out)
	}
	// 4. 命令成功后再确认二进制已出现
	if ufwBin() == "" {
		log.Printf("防火墙在线安装结束但未找到 ufw manager=%s", kind)
		return errors.New("安装结束但未找到 ufw")
	}
	log.Printf("防火墙在线安装完成 manager=%s", kind)
	return nil
}

/**
 * InstallOffline 安装面板写入临时目录的 deb 或 rpm。
 */
func (m *ufwManager) InstallOffline(path string) error {
	// 1. 只接受临时目录中本面板创建的包，避免执行任意路径
	if !safePkgPath(path) {
		return errors.New("安装包路径无效")
	}
	ext := strings.ToLower(filepath.Ext(path))
	if err := packageMagic(path, ext); err != nil {
		return err
	}
	// 2. 按扩展名调用对应安装器，不走 shell
	env := []string{"DEBIAN_FRONTEND=noninteractive"}
	var out string
	var err error
	switch ext {
	case ".deb":
		if !hasBin("dpkg") {
			return errors.New("未找到 dpkg")
		}
		out, err = runRoot(3*time.Minute, env, "dpkg", "-i", path)
	case ".rpm":
		if !hasBin("rpm") {
			return errors.New("未找到 rpm")
		}
		out, err = runRoot(3*time.Minute, nil, "rpm", "-Uvh", "--nosignature", path)
	default:
		return errors.New("不支持的安装包类型")
	}
	if err != nil {
		log.Printf("防火墙离线安装失败 ext=%s err=%s", ext, usefulLine(out))
		return publicInstallErr(out)
	}
	// 3. 安装后再确认 ufw 可用；依赖未装上时包管理器会先失败
	if ufwBin() == "" {
		log.Printf("防火墙离线安装结束但未找到 ufw ext=%s", ext)
		return errors.New("安装结束但未找到 ufw")
	}
	log.Printf("防火墙离线安装完成 ext=%s", ext)
	return nil
}

func safePkgPath(path string) bool {
	if path == "" || !filepath.IsAbs(path) {
		return false
	}
	base := filepath.Base(path)
	if !strings.HasPrefix(base, "agentdeck-fw-") {
		return false
	}
	dir := filepath.Clean(os.TempDir())
	rel, err := filepath.Rel(dir, filepath.Clean(path))
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return false
	}
	return true
}

func packageMagic(path, ext string) error {
	f, err := os.Open(path)
	if err != nil {
		return errors.New("读取安装包失败")
	}
	defer f.Close()
	buf := make([]byte, 8)
	n, err := f.Read(buf)
	if err != nil || n < 4 {
		return errors.New("安装包内容无效")
	}
	switch ext {
	case ".deb":
		if n < 8 || string(buf[:8]) != "!<arch>\n" {
			return errors.New("不是有效的 deb 包")
		}
	case ".rpm":
		if buf[0] != 0xed || buf[1] != 0xab || buf[2] != 0xee || buf[3] != 0xdb {
			return errors.New("不是有效的 rpm 包")
		}
	default:
		return errors.New("不支持的安装包类型")
	}
	return nil
}

// runRoot 当前是 root 就直接执行；否则用免密 sudo，不接受密码输入。
func runRoot(timeout time.Duration, env []string, name string, args ...string) (string, error) {
	if os.Geteuid() == 0 {
		return runCtx(timeout, env, name, args...)
	}
	if !hasBin("sudo") {
		return "", errors.New("需要 root 权限")
	}
	sargs := []string{"-n"}
	if len(env) > 0 {
		keys := make([]string, 0, len(env))
		for _, item := range env {
			if i := strings.IndexByte(item, '='); i > 0 {
				keys = append(keys, item[:i])
			}
		}
		if len(keys) > 0 {
			sargs = append(sargs, "--preserve-env="+strings.Join(keys, ","))
		}
	}
	sargs = append(sargs, name)
	sargs = append(sargs, args...)
	out, err := runCtx(timeout, env, "sudo", sargs...)
	if err != nil && sudoNeedsPassword(out) {
		return out, errors.New("需要 root 权限或免密 sudo")
	}
	return out, err
}

func sudoNeedsPassword(out string) bool {
	low := strings.ToLower(out)
	return strings.Contains(low, "password is required") ||
		strings.Contains(low, "a password is required") ||
		strings.Contains(low, "sorry, try again")
}

func publicInstallErr(out string) error {
	line := stripPkgPath(usefulLine(out))
	if line == "" || line == "命令执行失败" {
		return errors.New("安装失败")
	}
	return errors.New(line)
}

func stripPkgPath(s string) string {
	fields := strings.Fields(s)
	for i, field := range fields {
		if strings.Contains(field, "agentdeck-fw-") {
			fields[i] = "安装包"
		}
	}
	return strings.Join(fields, " ")
}
