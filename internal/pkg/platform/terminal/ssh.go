package terminal

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHOpenOpts 打开交互式 SSH 会话。
type SSHOpenOpts struct {
	Host       string
	Port       int
	Username   string
	AuthType   string // password | private_key
	Secret     string
	Passphrase string
	Cols       int
	Rows       int
	IgnoreHost bool
}

type sshSession struct {
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
	once    sync.Once
	mu      sync.Mutex
}

/**
 * OpenSSH 建立 SSH 连接并申请伪终端。
 */
func OpenSSH(opts SSHOpenOpts) (Session, error) {
	// 1. 组装认证方式
	auth, err := buildAuth(opts.AuthType, opts.Secret, opts.Passphrase)
	if err != nil {
		return nil, err
	}
	hostKey := ssh.InsecureIgnoreHostKey()
	if !opts.IgnoreHost {
		// 未实现 known_hosts 时拒绝非忽略模式
		return nil, fmt.Errorf("当前仅支持忽略主机指纹模式，请在终端配置中开启")
	}
	cfg := &ssh.ClientConfig{
		User:            opts.Username,
		Auth:            auth,
		HostKeyCallback: hostKey,
		Timeout:         15 * time.Second,
	}
	addr := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}

	// 2. 开 session 并请求 PTY
	sess, err := client.NewSession()
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("创建 SSH 会话失败: %w", err)
	}
	cols, rows := opts.Cols, opts.Rows
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sess.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		_ = sess.Close()
		_ = client.Close()
		return nil, fmt.Errorf("申请伪终端失败: %w", err)
	}
	stdin, err := sess.StdinPipe()
	if err != nil {
		_ = sess.Close()
		_ = client.Close()
		return nil, err
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		_ = sess.Close()
		_ = client.Close()
		return nil, err
	}
	stderr, err := sess.StderrPipe()
	if err != nil {
		_ = sess.Close()
		_ = client.Close()
		return nil, err
	}
	if err := sess.Shell(); err != nil {
		_ = sess.Close()
		_ = client.Close()
		return nil, fmt.Errorf("启动远程 shell 失败: %w", err)
	}
	return &sshSession{client: client, session: sess, stdin: stdin, stdout: stdout, stderr: stderr}, nil
}

func (s *sshSession) Read(p []byte) (int, error) {
	return s.stdout.Read(p)
}

func (s *sshSession) Write(p []byte) (int, error) {
	return s.stdin.Write(p)
}

func (s *sshSession) Resize(cols, rows int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	return s.session.WindowChange(rows, cols)
}

func (s *sshSession) Close() error {
	var err error
	s.once.Do(func() {
		if s.session != nil {
			_ = s.session.Close()
		}
		if s.client != nil {
			err = s.client.Close()
		}
	})
	return err
}

// SSHRunOpts 非交互 SSH 执行。
type SSHRunOpts struct {
	Host       string
	Port       int
	Username   string
	AuthType   string
	Secret     string
	Passphrase string
	Command    string
	Timeout    time.Duration
}

/**
 * SSHRun 在远程一次性执行命令并收集输出。
 */
func SSHRun(opts SSHRunOpts) (stdout, stderr string, exitCode int, err error) {
	auth, err := buildAuth(opts.AuthType, opts.Secret, opts.Passphrase)
	if err != nil {
		return "", "", -1, err
	}
	cfg := &ssh.ClientConfig{
		User:            opts.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	addr := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return "", "", -1, fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer client.Close()

	sess, err := client.NewSession()
	if err != nil {
		return "", "", -1, err
	}
	defer sess.Close()

	var outBuf, errBuf bytes.Buffer
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf

	done := make(chan error, 1)
	go func() { done <- sess.Run(opts.Command) }()

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	select {
	case err := <-done:
		code := 0
		if err != nil {
			if ee, ok := err.(*ssh.ExitError); ok {
				code = ee.ExitStatus()
			} else {
				return outBuf.String(), errBuf.String(), -1, err
			}
		}
		return outBuf.String(), errBuf.String(), code, nil
	case <-time.After(timeout):
		_ = sess.Close()
		return outBuf.String(), errBuf.String() + "\n执行超时", -1, nil
	}
}

// ProbeOpts 仅做握手探测，不打开 shell。
type ProbeOpts struct {
	Host       string
	Port       int
	Username   string
	AuthType   string
	Secret     string
	Passphrase string
	Timeout    time.Duration
}

/**
 * ProbeSSH 用给定凭证拨号并立即断开，用于保存前测试连通性。
 */
func ProbeSSH(opts ProbeOpts) (time.Duration, error) {
	// 1. 组装认证，缺凭证直接失败，避免打到远端才报空密码
	auth, err := buildAuth(opts.AuthType, opts.Secret, opts.Passphrase)
	if err != nil {
		return 0, err
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	cfg := &ssh.ClientConfig{
		User:            opts.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}
	addr := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	// 2. 拨号成功即视为连通，立刻关闭，不保留会话
	start := time.Now()
	client, err := ssh.Dial("tcp", addr, cfg)
	elapsed := time.Since(start)
	if err != nil {
		return elapsed, fmt.Errorf("连接失败: %w", err)
	}
	_ = client.Close()
	return elapsed, nil
}

func buildAuth(authType, secret, passphrase string) ([]ssh.AuthMethod, error) {
	switch authType {
	case "password":
		if secret == "" {
			return nil, fmt.Errorf("密码为空")
		}
		return []ssh.AuthMethod{ssh.Password(secret)}, nil
	case "private_key":
		var signer ssh.Signer
		var err error
		if passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(secret), []byte(passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(secret))
		}
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	default:
		return nil, fmt.Errorf("不支持的认证类型")
	}
}
