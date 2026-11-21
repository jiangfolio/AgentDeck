package secret

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jiangbyte/agentdeck/internal/config"
	"github.com/jiangbyte/agentdeck/internal/pkg/sm"
)

var (
	mu     sync.Mutex
	keyHex string
)

/**
 * EnsureKey 确保 data 目录存在本地 SM4 密钥文件，用于主机凭证落库加密。
 */
func EnsureKey() error {
	mu.Lock()
	defer mu.Unlock()
	if keyHex != "" {
		return nil
	}
	// 1. 准备目录与密钥路径
	dir := config.DataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}
	path := filepath.Join(dir, "term_secret.key")

	// 2. 已有则加载；否则生成 16 字节随机密钥
	if b, err := os.ReadFile(path); err == nil {
		k := string(b)
		if len(k) == 32 {
			keyHex = k
			return nil
		}
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Errorf("生成密钥失败: %w", err)
	}
	k := hex.EncodeToString(raw)
	if err := os.WriteFile(path, []byte(k), 0o600); err != nil {
		return fmt.Errorf("写入密钥失败: %w", err)
	}
	keyHex = k
	return nil
}

/**
 * Encrypt 将明文加密为十六进制密文；空串原样返回。
 */
func Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	if err := EnsureKey(); err != nil {
		return "", err
	}
	mu.Lock()
	k := keyHex
	mu.Unlock()
	return sm.SM4EncryptECB([]byte(plain), k)
}

/**
 * Decrypt 解密密文；空串原样返回。
 */
func Decrypt(cipherHex string) (string, error) {
	if cipherHex == "" {
		return "", nil
	}
	if err := EnsureKey(); err != nil {
		return "", err
	}
	mu.Lock()
	k := keyHex
	mu.Unlock()
	b, err := sm.SM4DecryptECB(cipherHex, k)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
