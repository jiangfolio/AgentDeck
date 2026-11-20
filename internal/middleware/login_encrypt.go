package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/config"
	"github.com/jiangbyte/agentdeck/internal/modular/crypto/key"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

const ctxSM4Key = "sm4_key_hex"

// LoginEncrypt 仅对登录接口做 SM4 加解密；其余 API 明文 JSON。
// 约定：X-SM4-Key = SM2Encrypt(UTF-8(sm4KeyHex))；Body = {"data": sm4CipherHex}。
func LoginEncrypt(svc *key.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 非登录路径直接放行，日常走明文 JSON
		if c.Request.URL.Path != config.LoginPath {
			c.Next()
			return
		}

		// 2. SM2 解密得到 32 位 hex 的 SM4 密钥
		encKey := strings.TrimSpace(c.GetHeader(config.HeaderSM4Key))
		if encKey == "" {
			response.AbortFail(c, http.StatusBadRequest, 400, "缺少加密密钥")
			return
		}
		sm4KeyBytes, err := svc.DecryptSM2Hex(encKey)
		if err != nil {
			response.AbortFail(c, http.StatusBadRequest, 400, "解密密钥失败")
			return
		}
		sm4KeyHex := strings.TrimSpace(string(sm4KeyBytes))
		if !isHexKey(sm4KeyHex, 32) {
			response.AbortFail(c, http.StatusBadRequest, 400, "会话密钥格式无效")
			return
		}
		c.Set(ctxSM4Key, sm4KeyHex)

		// 3. 解密请求体为明文 JSON
		if c.Request.Body != nil {
			raw, err := io.ReadAll(c.Request.Body)
			_ = c.Request.Body.Close()
			if err != nil {
				response.AbortFail(c, http.StatusBadRequest, 400, "读取请求失败")
				return
			}
			trimmed := strings.TrimSpace(string(raw))
			if trimmed == "" {
				response.AbortFail(c, http.StatusBadRequest, 400, "登录请求体为空")
				return
			}
			cipherHex := extractCipherHex(raw, trimmed)
			plain, err := svc.DecryptSM4(cipherHex, sm4KeyHex)
			if err != nil {
				response.AbortFail(c, http.StatusBadRequest, 400, "解密请求失败")
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(plain))
			c.Request.ContentLength = int64(len(plain))
			c.Request.Header.Set("Content-Type", "application/json")
		}

		// 4. 捕获 handler 响应并用同一 SM4 密钥加密
		bw := &bodyWriter{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = bw
		c.Next()

		plainResp := bw.buf.Bytes()
		if len(plainResp) == 0 {
			bw.ResponseWriter.WriteHeader(bw.status)
			return
		}
		cipherHex, err := svc.EncryptSM4(plainResp, sm4KeyHex)
		if err != nil {
			c.Writer = bw.ResponseWriter
			response.AbortFail(c, http.StatusInternalServerError, 500, "加密响应失败")
			return
		}
		out, _ := json.Marshal(map[string]string{"data": cipherHex})
		bw.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
		bw.ResponseWriter.Header().Del("Content-Length")
		bw.ResponseWriter.WriteHeader(bw.status)
		_, _ = bw.ResponseWriter.Write(out)
	}
}

func extractCipherHex(raw []byte, trimmed string) string {
	var wrap struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrap); err == nil && wrap.Data != "" {
		return wrap.Data
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil && asString != "" {
		return asString
	}
	return trimmed
}

type bodyWriter struct {
	gin.ResponseWriter
	buf    bytes.Buffer
	status int
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *bodyWriter) WriteString(s string) (int, error) {
	return w.buf.WriteString(s)
}

func (w *bodyWriter) WriteHeader(statusCode int) {
	w.status = statusCode
}

func isHexKey(s string, wantLen int) bool {
	if len(s) != wantLen {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
