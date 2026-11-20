package login

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jiangbyte/agentdeck/internal/modular/auth/session"
	"github.com/jiangbyte/agentdeck/internal/modular/auth/user"
	"github.com/jiangbyte/agentdeck/internal/config"
	"github.com/jiangbyte/agentdeck/internal/pkg/sm"
)

// Service 登录、登出与当前用户解析。
type Service struct {
	users    *user.Repo
	sessions *session.Repo
}

func NewService(users *user.Repo, sessions *session.Repo) *Service {
	return &Service{users: users, sessions: sessions}
}

// EnsureAdmin 首次启动写入默认管理员。
func (s *Service) EnsureAdmin() error {
	n, err := s.users.Count()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	salt, err := sm.RandomHex(16)
	if err != nil {
		return err
	}
	u := &user.User{
		Username:     config.DefaultAdmin,
		Salt:         salt,
		PasswordHash: sm.HashPassword(config.DefaultPass, salt),
		DisplayName:  "管理员",
	}
	if err := s.users.Create(u); err != nil {
		return fmt.Errorf("创建默认管理员失败: %w", err)
	}
	log.Printf("已创建默认管理员账号: %s", config.DefaultAdmin)
	return nil
}

// Result 登录成功返回值。
type Result struct {
	Token string
	User  *user.User
}

// Login 校验用户名密码并创建会话。
func (s *Service) Login(username, password string) (*Result, error) {
	// 1. 基本校验
	if username == "" || password == "" {
		return nil, errors.New("用户名或密码不能为空")
	}
	// 2. 查用户并比对 SM3 摘要
	u, err := s.users.FindByUsername(username)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("用户名或密码错误")
	}
	if sm.HashPassword(password, u.Salt) != u.PasswordHash {
		return nil, errors.New("用户名或密码错误")
	}
	// 3. 生成 opaque token 并写入 session
	token, err := sm.RandomHex(32)
	if err != nil {
		return nil, err
	}
	sess := &session.Session{
		UserID:    u.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(config.SessionTTL),
	}
	if err := s.sessions.Create(sess); err != nil {
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}
	return &Result{Token: token, User: u}, nil
}

// Logout 删除会话。
func (s *Service) Logout(token string) error {
	return s.sessions.DeleteByToken(token)
}

// CurrentUser 根据 session token 取当前用户。
func (s *Service) CurrentUser(token string) (*user.User, error) {
	if token == "" {
		return nil, nil
	}
	sess, err := s.sessions.FindValidByToken(token)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, nil
	}
	return s.users.FindByID(sess.UserID)
}

// DeleteExpiredSessions 清理过期会话。
func (s *Service) DeleteExpiredSessions() error {
	return s.sessions.DeleteExpired()
}
