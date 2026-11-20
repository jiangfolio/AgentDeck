package sm

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/sm3"
	"github.com/tjfoc/gmsm/sm4"
)

// SM2KeyPair 十六进制编码的 SM2 密钥对（与 sm-crypto 互通）。
type SM2KeyPair struct {
	PrivateKeyHex string
	PublicKeyHex  string
	PrivateKey    *sm2.PrivateKey
}

// GenerateSM2KeyPair 生成新的 SM2 密钥对。
func GenerateSM2KeyPair() (*SM2KeyPair, error) {
	priv, err := sm2.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("生成 SM2 密钥失败: %w", err)
	}
	return keyPairFromPrivate(priv), nil
}

// LoadSM2KeyPair 从十六进制私钥恢复密钥对。
func LoadSM2KeyPair(privateKeyHex string) (*SM2KeyPair, error) {
	// 1. 解析私钥标量
	d, ok := new(big.Int).SetString(strings.TrimSpace(privateKeyHex), 16)
	if !ok {
		return nil, errors.New("私钥格式无效")
	}
	// 2. 还原曲线点并组装私钥对象
	curve := sm2.P256Sm2()
	priv := new(sm2.PrivateKey)
	priv.Curve = curve
	priv.D = d
	priv.PublicKey.Curve = curve
	priv.PublicKey.X, priv.PublicKey.Y = curve.ScalarBaseMult(d.Bytes())
	return keyPairFromPrivate(priv), nil
}

func keyPairFromPrivate(priv *sm2.PrivateKey) *SM2KeyPair {
	pubHex := "04" + hex.EncodeToString(priv.PublicKey.X.FillBytes(make([]byte, 32))) +
		hex.EncodeToString(priv.PublicKey.Y.FillBytes(make([]byte, 32)))
	return &SM2KeyPair{
		PrivateKeyHex: hex.EncodeToString(priv.D.FillBytes(make([]byte, 32))),
		PublicKeyHex:  pubHex,
		PrivateKey:    priv,
	}
}

// SM2DecryptHex 使用 C1C3C2 模式解密 sm-crypto 产出的密文。
func (k *SM2KeyPair) SM2DecryptHex(cipherHex string) ([]byte, error) {
	if k == nil || k.PrivateKey == nil {
		return nil, errors.New("密钥未初始化")
	}
	cipherHex = strings.TrimSpace(cipherHex)
	if cipherHex == "" {
		return nil, errors.New("密文为空")
	}
	raw, err := hex.DecodeString(cipherHex)
	if err != nil {
		return nil, fmt.Errorf("密文非十六进制: %w", err)
	}
	// sm-crypto 输出为无 04 前缀的 C1C3C2；gmsm Decrypt 要求首字节为 0x04
	if len(raw) == 0 || raw[0] != 0x04 {
		raw = append([]byte{0x04}, raw...)
	}
	plain, err := sm2.Decrypt(k.PrivateKey, raw, sm2.C1C3C2)
	if err != nil {
		return nil, fmt.Errorf("SM2 解密失败: %w", err)
	}
	return plain, nil
}

// SM3Hash 计算 SM3 摘要并返回十六进制。
func SM3Hash(data []byte) string {
	sum := sm3.Sm3Sum(data)
	return hex.EncodeToString(sum)
}

// HashPassword 使用 salt + password 做 SM3 摘要。
func HashPassword(password, salt string) string {
	return SM3Hash([]byte(salt + password))
}

// RandomHex 生成 n 字节随机数的十六进制表示。
func RandomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// SM4EncryptECB 使用 SM4-ECB/PKCS7 加密（与 sm-crypto 默认模式对齐）。
func SM4EncryptECB(plain []byte, keyHex string) (string, error) {
	k, err := decodeSM4Key(keyHex)
	if err != nil {
		return "", err
	}
	out, err := sm4.Sm4Ecb(k, plain, true)
	if err != nil {
		return "", fmt.Errorf("SM4 加密失败: %w", err)
	}
	return hex.EncodeToString(out), nil
}

// SM4DecryptECB 使用 SM4-ECB/PKCS7 解密。
func SM4DecryptECB(cipherHex string, keyHex string) ([]byte, error) {
	k, err := decodeSM4Key(keyHex)
	if err != nil {
		return nil, err
	}
	raw, err := hex.DecodeString(strings.TrimSpace(cipherHex))
	if err != nil {
		return nil, fmt.Errorf("SM4 密文非十六进制: %w", err)
	}
	out, err := sm4.Sm4Ecb(k, raw, false)
	if err != nil {
		return nil, fmt.Errorf("SM4 解密失败: %w", err)
	}
	return out, nil
}

func decodeSM4Key(keyHex string) ([]byte, error) {
	keyHex = strings.TrimSpace(keyHex)
	k, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("SM4 密钥非十六进制: %w", err)
	}
	if len(k) != 16 {
		return nil, fmt.Errorf("SM4 密钥长度必须为 16 字节，当前 %d", len(k))
	}
	return k, nil
}
