package key

import (
	"github.com/gin-gonic/gin"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

// Handler 国密公钥下发。
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// PublicKey 明文返回 SM2 公钥，供登录协商会话密钥。
func (h *Handler) PublicKey(c *gin.Context) {
	pub := h.svc.PublicKeyHex()
	if pub == "" {
		response.Fail(c, 500, "公钥未就绪")
		return
	}
	response.OK(c, gin.H{"public_key": pub})
}
