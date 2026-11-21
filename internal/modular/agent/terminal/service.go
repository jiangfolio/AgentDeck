package terminal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jiangbyte/agentdeck/internal/middleware"
	"github.com/jiangbyte/agentdeck/internal/modular/agent/termhost"
	platterm "github.com/jiangbyte/agentdeck/internal/pkg/platform/terminal"
	"github.com/jiangbyte/agentdeck/internal/pkg/response"
)

const maxSessionsPerUser = 16

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // 同面板私用；Cookie 会话仍校验
	},
}

// Service 终端 WebSocket 会话。
type Service struct {
	hosts *termhost.Service
	mu    sync.Mutex
	// userID -> active count
	counts map[int64]int
}

func NewService(hosts *termhost.Service) *Service {
	return &Service{hosts: hosts, counts: map[int64]int{}}
}

type wsCtrl struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

/**
 * HandleWS 升级为 WebSocket，桥接本机 PTY 或 SSH。
 * query: kind=local|ssh, host_id, cols, rows, ignore_host=1|0
 */
func (s *Service) HandleWS(c *gin.Context) {
	u := middleware.CurrentUser(c)
	if u == nil {
		response.Fail(c, 401, "未登录")
		return
	}
	kind := strings.TrimSpace(c.Query("kind"))
	if kind == "" {
		kind = "local"
	}
	cols, _ := strconv.Atoi(c.DefaultQuery("cols", "120"))
	rows, _ := strconv.Atoi(c.DefaultQuery("rows", "30"))
	hostID := strings.TrimSpace(c.Query("host_id"))
	ignoreHost := c.DefaultQuery("ignore_host", "1") != "0"

	if !s.tryAcquire(u.ID) {
		response.Fail(c, 429, "终端会话数已达上限")
		return
	}
	released := false
	release := func() {
		if !released {
			released = true
			s.release(u.ID)
		}
	}

	// 1. 打开后端会话
	sess, err := s.openSession(kind, hostID, cols, rows, ignoreHost)
	if err != nil {
		release()
		response.Fail(c, 400, err.Error())
		return
	}

	// 2. 升级 WS
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		_ = sess.Close()
		release()
		return
	}
	defer func() {
		_ = conn.Close()
		_ = sess.Close()
		release()
	}()

	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Minute))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Minute))
		return nil
	})

	done := make(chan struct{})
	var once sync.Once
	closeDone := func() { once.Do(func() { close(done) }) }

	// 3. PTY/SSH → WS
	go func() {
		defer closeDone()
		buf := make([]byte, 4096)
		for {
			n, err := sess.Read(buf)
			if n > 0 {
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// 4. WS → PTY/SSH（文本为控制 JSON，二进制/其它为输入）
	go func() {
		defer closeDone()
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.TextMessage && len(data) > 0 && data[0] == '{' {
				var ctrl wsCtrl
				if json.Unmarshal(data, &ctrl) == nil && ctrl.Type == "resize" {
					_ = sess.Resize(ctrl.Cols, ctrl.Rows)
					continue
				}
			}
			if _, err := sess.Write(data); err != nil {
				return
			}
		}
	}()

	<-done
}

func (s *Service) openSession(kind, hostID string, cols, rows int, ignoreHost bool) (platterm.Session, error) {
	switch kind {
	case "local":
		return platterm.OpenLocal(uint16(cols), uint16(rows))
	case "ssh":
		if hostID == "" {
			return nil, fmt.Errorf("缺少 host_id")
		}
		h, secret, pass, err := s.hosts.DecryptAuth(hostID)
		if err != nil {
			return nil, err
		}
		return platterm.OpenSSH(platterm.SSHOpenOpts{
			Host: h.Host, Port: h.Port, Username: h.Username, AuthType: h.AuthType,
			Secret: secret, Passphrase: pass, Cols: cols, Rows: rows, IgnoreHost: ignoreHost,
		})
	default:
		return nil, fmt.Errorf("未知会话类型")
	}
}

func (s *Service) tryAcquire(userID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.counts[userID] >= maxSessionsPerUser {
		return false
	}
	s.counts[userID]++
	return true
}

func (s *Service) release(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.counts[userID] > 0 {
		s.counts[userID]--
	}
	if s.counts[userID] == 0 {
		delete(s.counts, userID)
	}
}
