package termhost

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	platterm "github.com/jiangbyte/agentdeck/internal/pkg/platform/terminal"
	"github.com/jiangbyte/agentdeck/internal/pkg/secret"
	"gorm.io/gorm"
)

// Service 主机与分组业务。
type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

/**
 * ListGroups 返回全部分组（分组量小，供下拉；仍走仓储有序查询）。
 */
func (s *Service) ListGroups() ([]HostGroup, error) {
	return s.repo.ListGroups()
}

/**
 * SaveGroup 创建或更新分组。
 */
func (s *Service) SaveGroup(req SaveGroupRequest) (*HostGroup, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("分组名称不能为空")
	}
	if strings.TrimSpace(req.ID) == "" {
		g := &HostGroup{ID: uuid.NewString(), Name: name, Sort: req.Sort, CreatedAt: time.Now()}
		if err := s.repo.CreateGroup(g); err != nil {
			return nil, fmt.Errorf("创建分组失败")
		}
		return g, nil
	}
	g, err := s.repo.FindGroup(req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("分组不存在")
		}
		return nil, err
	}
	g.Name = name
	g.Sort = req.Sort
	if err := s.repo.UpdateGroup(g); err != nil {
		return nil, fmt.Errorf("更新分组失败")
	}
	return g, nil
}

/**
 * RemoveGroup 删除空分组；仍有主机时拒绝。
 */
func (s *Service) RemoveGroup(id string) error {
	n, err := s.repo.CountHostsByGroup(id)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("分组下仍有主机，无法删除")
	}
	return s.repo.DeleteGroup(id)
}

/**
 * List 分页列出主机并附带分组名。
 */
func (s *Service) List(q ListQuery) (*ListResult, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	items, total, err := s.repo.ListHosts(q.Page, q.PageSize, q.Keyword, q.GroupID)
	if err != nil {
		return nil, err
	}
	gids := make([]string, 0, len(items))
	for _, h := range items {
		gids = append(gids, h.GroupID)
	}
	names, err := s.repo.MapGroupNames(gids)
	if err != nil {
		return nil, err
	}
	views := make([]HostView, 0, len(items))
	for _, h := range items {
		views = append(views, toView(h, names[h.GroupID]))
	}
	return &ListResult{Items: views, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

/**
 * Save 创建或更新主机；空 secret/passphrase 表示保留原密文。
 */
func (s *Service) Save(req SaveHostRequest) (*HostView, error) {
	// 1. 规范化与基础校验
	title := strings.TrimSpace(req.Title)
	hostAddr := strings.TrimSpace(req.Host)
	user := strings.TrimSpace(req.Username)
	if title == "" || hostAddr == "" || user == "" {
		return nil, fmt.Errorf("标题、地址、用户名不能为空")
	}
	if req.Port < 1 || req.Port > 65535 {
		return nil, fmt.Errorf("端口无效")
	}
	auth := req.AuthType
	if auth != "password" && auth != "private_key" {
		return nil, fmt.Errorf("认证类型无效")
	}
	if req.GroupID != "" {
		if _, err := s.repo.FindGroup(req.GroupID); err != nil {
			return nil, fmt.Errorf("分组不存在")
		}
	}

	now := time.Now()
	if strings.TrimSpace(req.ID) == "" {
		// 2. 新建：必须提供凭证
		if strings.TrimSpace(req.Secret) == "" {
			return nil, fmt.Errorf("请填写密码或私钥")
		}
		sec, err := secret.Encrypt(req.Secret)
		if err != nil {
			return nil, fmt.Errorf("加密凭证失败")
		}
		passEnc, err := secret.Encrypt(req.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("加密口令失败")
		}
		h := &Host{
			ID: uuid.NewString(), GroupID: req.GroupID, Title: title, Host: hostAddr, Port: req.Port,
			Username: user, AuthType: auth, SecretEnc: sec, PassphraseEnc: passEnc,
			Description: strings.TrimSpace(req.Description), Sort: req.Sort, CreatedAt: now, UpdatedAt: now,
		}
		if err := s.repo.CreateHost(h); err != nil {
			return nil, fmt.Errorf("创建主机失败")
		}
		return s.viewOf(h)
	}

	// 3. 更新：按需改密文列
	h, err := s.repo.FindHost(req.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("主机不存在")
		}
		return nil, err
	}
	h.GroupID = req.GroupID
	h.Title = title
	h.Host = hostAddr
	h.Port = req.Port
	h.Username = user
	h.AuthType = auth
	h.Description = strings.TrimSpace(req.Description)
	h.Sort = req.Sort
	h.UpdatedAt = now
	cols := []string{"group_id", "title", "host", "port", "username", "auth_type", "description", "sort", "updated_at"}
	if strings.TrimSpace(req.Secret) != "" {
		sec, err := secret.Encrypt(req.Secret)
		if err != nil {
			return nil, fmt.Errorf("加密凭证失败")
		}
		h.SecretEnc = sec
		cols = append(cols, "secret_enc")
	}
	if req.Passphrase != "" || req.AuthType == "password" {
		// 切到密码认证时清空口令；显式传口令则更新
		if req.AuthType == "password" {
			h.PassphraseEnc = ""
			cols = append(cols, "passphrase_enc")
		} else if req.Passphrase != "" {
			pe, err := secret.Encrypt(req.Passphrase)
			if err != nil {
				return nil, fmt.Errorf("加密口令失败")
			}
			h.PassphraseEnc = pe
			cols = append(cols, "passphrase_enc")
		}
	}
	if err := s.repo.UpdateHost(h, cols); err != nil {
		return nil, fmt.Errorf("更新主机失败")
	}
	return s.viewOf(h)
}

/**
 * Remove 批量删除主机。
 */
func (s *Service) Remove(ids []string) error {
	clean := uniqueNonEmpty(ids)
	if len(clean) == 0 {
		return fmt.Errorf("未选择主机")
	}
	return s.repo.DeleteHosts(clean)
}

/**
 * Test 用表单凭证探测 SSH。未改密时回退到已保存密文，响应里不带凭证。
 */
func (s *Service) Test(req TestRequest) (*TestResult, error) {
	// 1. 优先用本次填写的密码/私钥；留空则取已存主机
	sec := req.Secret
	pass := req.Passphrase
	auth := req.AuthType
	if strings.TrimSpace(sec) == "" {
		if strings.TrimSpace(req.ID) == "" {
			return nil, fmt.Errorf("请填写密码或私钥后再测试")
		}
		_, stored, storedPass, err := s.DecryptAuth(req.ID)
		if err != nil {
			return nil, err
		}
		sec = stored
		if pass == "" {
			pass = storedPass
		}
	}
	// 2. 拨号探测，失败也返回可读结果，便于表单直接展示
	elapsed, err := platterm.ProbeSSH(platterm.ProbeOpts{
		Host: strings.TrimSpace(req.Host), Port: req.Port, Username: strings.TrimSpace(req.Username),
		AuthType: auth, Secret: sec, Passphrase: pass, Timeout: 12 * time.Second,
	})
	res := &TestResult{LatencyMs: elapsed.Milliseconds()}
	if err != nil {
		res.OK = false
		res.Message = err.Error()
		return res, nil
	}
	res.OK = true
	res.Message = fmt.Sprintf("连接成功，耗时 %d ms", res.LatencyMs)
	return res, nil
}

/**
 * DecryptAuth 供终端 SSH 使用：解密凭证，勿写入日志或 API 响应。
 */
func (s *Service) DecryptAuth(id string) (*Host, string, string, error) {
	h, err := s.repo.FindHost(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, "", "", fmt.Errorf("主机不存在")
		}
		return nil, "", "", err
	}
	sec, err := secret.Decrypt(h.SecretEnc)
	if err != nil {
		return nil, "", "", fmt.Errorf("解密凭证失败")
	}
	pass, err := secret.Decrypt(h.PassphraseEnc)
	if err != nil {
		return nil, "", "", fmt.Errorf("解密口令失败")
	}
	return h, sec, pass, nil
}

func (s *Service) viewOf(h *Host) (*HostView, error) {
	name := ""
	if h.GroupID != "" {
		m, err := s.repo.MapGroupNames([]string{h.GroupID})
		if err != nil {
			return nil, err
		}
		name = m[h.GroupID]
	}
	v := toView(*h, name)
	return &v, nil
}

func toView(h Host, groupName string) HostView {
	return HostView{
		ID: h.ID, GroupID: h.GroupID, GroupName: groupName, Title: h.Title, Host: h.Host, Port: h.Port,
		Username: h.Username, AuthType: h.AuthType, HasSecret: h.SecretEnc != "", Description: h.Description,
		Sort: h.Sort, CreatedAt: h.CreatedAt, UpdatedAt: h.UpdatedAt,
	}
}
