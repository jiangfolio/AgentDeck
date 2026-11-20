package firewall

import (
	"errors"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	plat "github.com/jiangbyte/agentdeck/internal/pkg/platform/firewall"
)

const maxFirewallPkg = 64 << 20

// Service 防火墙读写。写操作串行，避免并发改规则。
type Service struct {
	mu  sync.Mutex
	mgr plat.Manager
}

func NewService() *Service {
	return &Service{mgr: plat.New()}
}

func (s *Service) Status() plat.Status {
	return s.mgr.Status()
}

/**
 * Rules 关键字过滤后分页。读取失败时返回空列表和说明，不让整页失败。
 */
func (s *Service) Rules(q RulesQuery) *RulesResult {
	page, size := normPage(q.Page, q.PageSize)
	rules, err := s.mgr.Rules()
	msg := ""
	if err != nil {
		msg = err.Error()
		rules = []plat.Rule{}
	}
	st := s.mgr.Status()
	if msg == "" {
		msg = st.Message
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	filtered := make([]plat.Rule, 0, len(rules))
	for _, r := range rules {
		if kw == "" ||
			strings.Contains(strings.ToLower(r.IP), kw) ||
			strings.Contains(strings.ToLower(r.Port), kw) ||
			strings.Contains(strings.ToLower(r.Protocol), kw) ||
			strings.Contains(strings.ToLower(r.Description), kw) ||
			strings.Contains(strings.ToLower(r.Action), kw) {
			filtered = append(filtered, r)
		}
	}
	total := len(filtered)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	items := filtered[start:end]
	if items == nil {
		items = []plat.Rule{}
	}
	return &RulesResult{Items: items, Total: total, Page: page, PageSize: size, Message: msg}
}

func (s *Service) SetEnabled(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mgr.SetEnabled(enabled)
}

/**
 * Save 校验后创建或按同一 id 更新规则。
 */
func (s *Service) Save(req SaveRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(req)
}

func (s *Service) saveLocked(req SaveRequest) error {
	rule := plat.Rule{
		ID:          strings.TrimSpace(req.ID),
		Action:      strings.ToLower(strings.TrimSpace(req.Action)),
		Protocol:    strings.ToLower(strings.TrimSpace(req.Protocol)),
		IP:          strings.TrimSpace(req.IP),
		Port:        strings.TrimSpace(req.Port),
		Description: strings.TrimSpace(req.Description),
	}
	if rule.Protocol == "" {
		rule.Protocol = "any"
	}
	if err := validate(rule); err != nil {
		return err
	}
	if rule.ID == "" {
		rule.ID = strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	}
	return s.mgr.Save(rule)
}

func (s *Service) Remove(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("规则不存在")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mgr.Remove(id)
}

func (s *Service) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mgr.Reset()
}

/**
 * InstallOnline 按当前平台在线安装防火墙组件。已安装时直接返回。
 */
func (s *Service) InstallOnline() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 1. 已安装或不支持在线安装时不再执行包管理器
	st := s.mgr.Status()
	if st.Installed {
		return nil
	}
	if !st.CanOnline {
		return errors.New("当前平台不能在线安装")
	}
	// 2. 安装过程持锁，避免同时改规则
	return s.mgr.InstallOnline()
}

/**
 * InstallOffline 校验上传的安装包后交给平台安装器。
 */
func (s *Service) InstallOffline(name string, r io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 1. 已安装或不支持离线时直接拒绝，避免把文件交给系统安装器
	if r == nil {
		return errors.New("请上传安装包")
	}
	st := s.mgr.Status()
	if st.Installed {
		return errors.New("防火墙组件已安装")
	}
	if !st.CanOffline {
		return errors.New("当前平台不支持离线安装")
	}
	ext, ok := matchOfflineExt(name, st.OfflineExt)
	if !ok {
		return errors.New("不支持的安装包类型")
	}
	// 2. 只把允许的扩展名写入临时文件，并限制大小
	f, err := os.CreateTemp("", "agentdeck-fw-*"+ext)
	if err != nil {
		log.Printf("防火墙离线安装创建临时文件失败: %v", err)
		return errors.New("保存安装包失败")
	}
	path := f.Name()
	defer os.Remove(path)
	n, copyErr := io.Copy(f, io.LimitReader(r, maxFirewallPkg+1))
	closeErr := f.Close()
	if copyErr != nil || closeErr != nil {
		log.Printf("防火墙离线安装写入失败: copy=%v close=%v", copyErr, closeErr)
		return errors.New("保存安装包失败")
	}
	if n <= 0 || n > maxFirewallPkg {
		return errors.New("安装包大小无效")
	}
	// 3. 平台侧再核对文件头并调用包管理器，结束后删除临时文件
	return s.mgr.InstallOffline(path)
}

func matchOfflineExt(name string, allowed []string) (string, bool) {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(name)))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "", false
	}
	best := ""
	for _, ext := range allowed {
		ext = strings.ToLower(strings.TrimSpace(ext))
		if ext == "" || !strings.HasPrefix(ext, ".") || !strings.HasSuffix(base, ext) {
			continue
		}
		if len(ext) > len(best) {
			best = ext
		}
	}
	if best == "" {
		return "", false
	}
	return best, true
}

func (s *Service) Export() ([]plat.Rule, error) {
	return s.mgr.Rules()
}

/**
 * Import 合并时按协议+端口+IP+动作去重；替换时只删 AgentDeck 创建的规则。
 */
func (s *Service) Import(req ImportRequest) error {
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode != "merge" && mode != "replace" {
		return errors.New("导入模式无效")
	}
	if len(req.Rules) == 0 {
		return errors.New("没有可导入的规则")
	}
	for _, r := range req.Rules {
		item := plat.Rule{
			Action:      strings.ToLower(strings.TrimSpace(r.Action)),
			Protocol:    strings.ToLower(strings.TrimSpace(r.Protocol)),
			IP:          strings.TrimSpace(r.IP),
			Port:        strings.TrimSpace(r.Port),
			Description: strings.TrimSpace(r.Description),
		}
		if item.Protocol == "" {
			item.Protocol = "any"
		}
		if err := validate(item); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, err := s.mgr.Rules()
	if err != nil {
		return err
	}
	if mode == "replace" {
		for _, r := range existing {
			if r.Managed {
				if err := s.mgr.Remove(r.ID); err != nil {
					return err
				}
			}
		}
		existing = nil
	}
	seen := map[string]struct{}{}
	for _, r := range existing {
		seen[ruleKey(r)] = struct{}{}
	}
	for _, r := range req.Rules {
		item := plat.Rule{
			Action:      strings.ToLower(strings.TrimSpace(r.Action)),
			Protocol:    strings.ToLower(strings.TrimSpace(r.Protocol)),
			IP:          strings.TrimSpace(r.IP),
			Port:        strings.TrimSpace(r.Port),
			Description: strings.TrimSpace(r.Description),
		}
		if item.Protocol == "" {
			item.Protocol = "any"
		}
		if _, ok := seen[ruleKey(item)]; ok {
			continue
		}
		item.ID = strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
		if err := s.mgr.Save(item); err != nil {
			return err
		}
		seen[ruleKey(item)] = struct{}{}
	}
	return nil
}

func validate(rule plat.Rule) error {
	if rule.Action != "allow" && rule.Action != "deny" {
		return errors.New("动作无效")
	}
	if rule.Protocol != "any" && rule.Protocol != "tcp" && rule.Protocol != "udp" {
		return errors.New("协议无效")
	}
	if strings.ContainsAny(rule.Description, "\r\n") || len([]rune(rule.Description)) > 128 {
		return errors.New("描述无效")
	}
	if rule.Port != "" && !validPort(rule.Port) {
		return errors.New("端口无效")
	}
	if rule.IP != "" && !strings.EqualFold(rule.IP, "any") && !validIP(rule.IP) {
		return errors.New("IP 无效")
	}
	return nil
}

func validPort(port string) bool {
	parts := strings.Split(port, ":")
	if len(parts) != 1 && len(parts) != 2 {
		return false
	}
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return false
		}
	}
	return true
}

func validIP(ip string) bool {
	if net.ParseIP(ip) != nil {
		return true
	}
	_, _, err := net.ParseCIDR(ip)
	return err == nil
}

func ruleKey(r plat.Rule) string {
	return strings.ToLower(r.Action + "|" + r.Protocol + "|" + r.Port + "|" + r.IP)
}

func normPage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
