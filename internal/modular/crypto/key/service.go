package key

import (
	"fmt"
	"log"
	"sync"

	"github.com/jiangbyte/agentdeck/internal/config"
	"github.com/jiangbyte/agentdeck/internal/pkg/sm"
)

// Service 管理服务端 SM2 密钥与加解密。
type Service struct {
	repo *Repo
	mu   sync.RWMutex
	pair *sm.SM2KeyPair
}

func NewService(r *Repo) *Service {
	return &Service{repo: r}
}

// EnsureKey 首次启动生成并持久化 SM2 密钥，后续直接加载。
func (s *Service) EnsureKey() error {
	// 1. 已有内存密钥则直接返回
	s.mu.RLock()
	if s.pair != nil {
		s.mu.RUnlock()
		return nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pair != nil {
		return nil
	}

	// 2. 尝试从数据库加载
	row, err := s.repo.FindByName(config.CryptoKeyName)
	if err != nil {
		return err
	}
	if row != nil {
		pair, err := sm.LoadSM2KeyPair(row.PrivateKey)
		if err != nil {
			return fmt.Errorf("加载 SM2 密钥失败: %w", err)
		}
		s.pair = pair
		log.Println("已加载服务端 SM2 密钥")
		return nil
	}

	// 3. 首次生成并入库
	pair, err := sm.GenerateSM2KeyPair()
	if err != nil {
		return err
	}
	if err := s.repo.Create(&Key{
		Name:       config.CryptoKeyName,
		PrivateKey: pair.PrivateKeyHex,
		PublicKey:  pair.PublicKeyHex,
	}); err != nil {
		return fmt.Errorf("保存 SM2 密钥失败: %w", err)
	}
	s.pair = pair
	log.Println("已生成并保存服务端 SM2 密钥")
	return nil
}

// PublicKeyHex 返回 SM2 公钥十六进制（含 04 前缀）。
func (s *Service) PublicKeyHex() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.pair == nil {
		return ""
	}
	return s.pair.PublicKeyHex
}

// DecryptSM2Hex 解密前端用公钥加密的数据。
func (s *Service) DecryptSM2Hex(cipherHex string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.pair == nil {
		return nil, fmt.Errorf("SM2 密钥未就绪")
	}
	return s.pair.SM2DecryptHex(cipherHex)
}

// EncryptSM4 / DecryptSM4 封装对称加解密。
func (s *Service) EncryptSM4(plain []byte, keyHex string) (string, error) {
	return sm.SM4EncryptECB(plain, keyHex)
}

func (s *Service) DecryptSM4(cipherHex, keyHex string) ([]byte, error) {
	return sm.SM4DecryptECB(cipherHex, keyHex)
}
