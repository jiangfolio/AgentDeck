package files

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/jiangbyte/agentdeck/internal/pkg/platform/archive"
	plat "github.com/jiangbyte/agentdeck/internal/pkg/platform/files"
)

const (
	maxEditBytes   = 2 << 20  // 文本编辑上限 2MB
	maxUploadBytes = 64 << 20 // 上传上限 64MB
)

// Service 本机文件浏览与读写。
type Service struct {
	mu sync.Mutex
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Roots() []string {
	roots := plat.Roots()
	if roots == nil {
		return []string{}
	}
	return roots
}

/**
 * List 列出目录内容，关键字过滤后分页。
 */
func (s *Service) List(q ListQuery) (*ListResult, error) {
	// 1. 规范化路径；空路径取第一个根
	path := strings.TrimSpace(q.Path)
	if path == "" {
		roots := s.Roots()
		if len(roots) == 0 {
			return nil, errors.New("没有可用的根路径")
		}
		path = roots[0]
	}
	var err error
	path, err = s.resolvePath(path)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil, publicFSErr(err)
	}
	if !st.IsDir() {
		return nil, errors.New("路径不是目录")
	}
	// 2. 读取目录项并过滤关键字
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, publicFSErr(err)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	items := make([]FileItem, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		full := filepath.Join(path, name)
		// 回收站目录不在普通列表里展示
		if isUnderTrash(full) {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(name), kw) {
			continue
		}
		info, iErr := e.Info()
		item := FileItem{
			Name:  name,
			Path:  full,
			IsDir: e.IsDir(),
		}
		if iErr == nil {
			item.Size = info.Size()
			item.Mode = formatMode(info)
			item.ModTime = info.ModTime().Format("2006-01-02 15:04:05")
			item.Owner = fileOwner(info)
			if e.IsDir() {
				item.Size = 0
			}
		}
		items = append(items, item)
	}
	// 3. 目录在前、同名按大小写不敏感排序
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	// 4. 分页裁剪
	page, size := normPage(q.Page, q.PageSize)
	total := len(items)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	pageItems := items[start:end]
	if pageItems == nil {
		pageItems = []FileItem{}
	}
	return &ListResult{
		Path:     path,
		Parent:   parentPath(path),
		Items:    pageItems,
		Total:    total,
		Page:     page,
		PageSize: size,
	}, nil
}

/**
 * Read 读取可编辑文本；过大或二进制拒绝。
 */
func (s *Service) Read(path string) (*ReadResult, error) {
	// 1. 校验为普通文件且体积允许
	abs, err := s.resolvePath(path)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, publicFSErr(err)
	}
	if st.IsDir() {
		return nil, errors.New("不能编辑目录")
	}
	if st.Size() > maxEditBytes {
		return nil, errors.New("文件超过 2MB，请下载后编辑")
	}
	// 2. 读入并判定是否为可显示文本
	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, publicFSErr(err)
	}
	if !isEditableText(raw) {
		return nil, errors.New("二进制或不支持的编码，请下载查看")
	}
	return &ReadResult{
		Path:     abs,
		Name:     filepath.Base(abs),
		Content:  string(raw),
		Size:     st.Size(),
		Language: guessLanguage(filepath.Base(abs)),
	}, nil
}

/**
 * Write 整文件覆盖写入文本。
 */
func (s *Service) Write(req WriteRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 1. 校验目标为已存在的普通文件
	abs, err := s.resolvePath(req.Path)
	if err != nil {
		return err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return publicFSErr(err)
	}
	if st.IsDir() {
		return errors.New("不能写入目录")
	}
	content := req.Content
	if int64(len(content)) > maxEditBytes {
		return errors.New("内容超过 2MB")
	}
	// 2. 先写临时文件再替换，降低半写入风险
	dir := filepath.Dir(abs)
	tmp, err := os.CreateTemp(dir, ".agentdeck-edit-*")
	if err != nil {
		return publicFSErr(err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := io.WriteString(tmp, content); err != nil {
		tmp.Close()
		return publicFSErr(err)
	}
	if err := tmp.Close(); err != nil {
		return publicFSErr(err)
	}
	if err := os.Chmod(tmpName, st.Mode()); err != nil {
		return publicFSErr(err)
	}
	if err := os.Rename(tmpName, abs); err != nil {
		return publicFSErr(err)
	}
	return nil
}

/**
 * Mkdir 在目录下新建子目录。
 */
func (s *Service) Mkdir(req NameRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	parent, name, err := s.resolveChild(req.Path, req.Name)
	if err != nil {
		return err
	}
	target := filepath.Join(parent, name)
	if err := os.Mkdir(target, 0o755); err != nil {
		return publicFSErr(err)
	}
	return nil
}

/**
 * Create 在目录下新建空文件。
 */
func (s *Service) Create(req NameRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	parent, name, err := s.resolveChild(req.Path, req.Name)
	if err != nil {
		return err
	}
	target := filepath.Join(parent, name)
	f, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return publicFSErr(err)
	}
	return f.Close()
}

/**
 * Rename 同目录重命名。
 */
func (s *Service) Rename(req RenameRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 1. 解析源路径并校验新名
	abs, err := s.resolvePath(req.Path)
	if err != nil {
		return err
	}
	newName := strings.TrimSpace(req.NewName)
	if err := validateName(newName); err != nil {
		return err
	}
	// 2. 目标必须落在同一父目录
	dest := filepath.Join(filepath.Dir(abs), newName)
	if _, err := os.Stat(dest); err == nil {
		return errors.New("目标已存在")
	}
	if err := os.Rename(abs, dest); err != nil {
		return publicFSErr(err)
	}
	return nil
}

/**
 * Remove 移入回收站，不直接永久删除。
 */
func (s *Service) Remove(paths []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(paths) == 0 {
		return errors.New("请选择要删除的项")
	}
	for _, p := range paths {
		abs, err := s.resolvePath(p)
		if err != nil {
			return err
		}
		if isRootPath(abs) {
			return errors.New("不能删除根路径")
		}
		if isUnderTrash(abs) {
			return errors.New("回收站内请使用彻底删除")
		}
		st, err := os.Stat(abs)
		if err != nil {
			return publicFSErr(err)
		}
		if err := moveToTrash(abs, st); err != nil {
			return err
		}
	}
	return nil
}

/**
 * Copy 复制文件或目录树到目标路径。
 */
func (s *Service) Copy(req CopyMoveRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	from, to, err := s.resolveCopyMove(req)
	if err != nil {
		return err
	}
	return copyPath(from, to)
}

/**
 * Move 移动文件或目录。
 */
func (s *Service) Move(req CopyMoveRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	from, to, err := s.resolveCopyMove(req)
	if err != nil {
		return err
	}
	if err := os.Rename(from, to); err != nil {
		// 跨设备时 rename 失败，退化为复制再删
		if err := copyPath(from, to); err != nil {
			return err
		}
		st, _ := os.Stat(from)
		if st != nil && st.IsDir() {
			return publicFSErr(os.RemoveAll(from))
		}
		return publicFSErr(os.Remove(from))
	}
	return nil
}

/**
 * Upload 把上传流写入目标目录下的文件名。
 */
func (s *Service) Upload(dir, filename string, r io.Reader, size int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 1. 校验目录与文件名、大小
	if r == nil {
		return errors.New("请上传文件")
	}
	if size < 0 || size > maxUploadBytes {
		return errors.New("上传文件不能超过 64MB")
	}
	parent, name, err := s.resolveChild(dir, filename)
	if err != nil {
		return err
	}
	target := filepath.Join(parent, name)
	// 2. 限流写入，避免超限内容落盘
	f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return publicFSErr(err)
	}
	defer f.Close()
	written, err := io.Copy(f, io.LimitReader(r, maxUploadBytes+1))
	if err != nil {
		_ = os.Remove(target)
		return publicFSErr(err)
	}
	if written > maxUploadBytes {
		_ = os.Remove(target)
		return errors.New("上传文件不能超过 64MB")
	}
	return nil
}

/**
 * OpenDownload 打开供下载的只读文件句柄。
 */
func (s *Service) OpenDownload(path string) (*os.File, os.FileInfo, error) {
	abs, err := s.resolvePath(path)
	if err != nil {
		return nil, nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, nil, publicFSErr(err)
	}
	if st.IsDir() {
		return nil, nil, errors.New("不能下载目录")
	}
	f, err := os.Open(abs)
	if err != nil {
		return nil, nil, publicFSErr(err)
	}
	return f, st, nil
}

/**
 * Extract 解压压缩包到同级目录（或指定 dest）；纯 Go 跨平台。
 */
func (s *Service) Extract(req ExtractRequest) (*ExtractResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 1. 校验源为支持的压缩文件
	abs, err := s.resolvePath(req.Path)
	if err != nil {
		return nil, err
	}
	if isUnderTrash(abs) {
		return nil, errors.New("请先还原后再解压")
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, publicFSErr(err)
	}
	if st.IsDir() {
		return nil, errors.New("请选择压缩包文件")
	}
	if !archive.IsArchive(abs) {
		return nil, errors.New("不支持的压缩格式，请使用 zip / tar / tar.gz / tar.xz")
	}
	// 2. 解析目标目录（默认同级去后缀目录名）
	dest := strings.TrimSpace(req.Dest)
	if dest == "" {
		dest = archive.DefaultExtractDir(abs)
	} else {
		dest, err = s.resolvePath(dest)
		if err != nil {
			return nil, err
		}
	}
	if err := archive.Extract(abs, dest); err != nil {
		return nil, err
	}
	return &ExtractResult{Dest: dest}, nil
}

/**
 * Compress 将选中路径打成 zip 或 tar.gz。
 */
func (s *Service) Compress(req CompressRequest) (*CompressResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 1. 解析并校验全部源路径
	if len(req.Paths) == 0 {
		return nil, errors.New("请选择要压缩的文件或目录")
	}
	sources := make([]string, 0, len(req.Paths))
	for _, p := range req.Paths {
		abs, err := s.resolvePath(p)
		if err != nil {
			return nil, err
		}
		if isRootPath(abs) || isUnderTrash(abs) {
			return nil, errors.New("不能压缩根路径或回收站内容")
		}
		if _, err := os.Stat(abs); err != nil {
			return nil, publicFSErr(err)
		}
		sources = append(sources, abs)
	}
	format := archive.Format(strings.ToLower(strings.TrimSpace(req.Format)))
	if format == "" {
		format = archive.FormatZIP
	}
	if format != archive.FormatZIP && format != archive.FormatTarGZ {
		return nil, errors.New("压缩仅支持 zip / tar.gz")
	}
	// 2. 确定输出路径
	dest := strings.TrimSpace(req.Dest)
	if dest == "" {
		name, err := archive.DefaultArchiveName(sources[0], format)
		if err != nil {
			return nil, err
		}
		if len(sources) > 1 {
			name = "archive"
			if format == archive.FormatZIP {
				name += ".zip"
			} else {
				name += ".tar.gz"
			}
		}
		dest = filepath.Join(filepath.Dir(sources[0]), name)
	}
	dest, err := s.resolvePath(dest)
	if err != nil {
		return nil, err
	}
	if err := archive.Compress(sources, dest, format); err != nil {
		return nil, err
	}
	return &CompressResult{Path: dest}, nil
}

func (s *Service) resolveCopyMove(req CopyMoveRequest) (string, string, error) {
	from, err := s.resolvePath(req.From)
	if err != nil {
		return "", "", err
	}
	to, err := s.resolvePath(req.To)
	if err != nil {
		return "", "", err
	}
	if from == to {
		return "", "", errors.New("源与目标相同")
	}
	if isRootPath(from) {
		return "", "", errors.New("不能操作根路径")
	}
	if _, err := os.Stat(from); err != nil {
		return "", "", publicFSErr(err)
	}
	if _, err := os.Stat(to); err == nil {
		return "", "", errors.New("目标已存在")
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return "", "", publicFSErr(err)
	}
	return from, to, nil
}

func (s *Service) resolveChild(dir, name string) (string, string, error) {
	parent, err := s.resolvePath(dir)
	if err != nil {
		return "", "", err
	}
	st, err := os.Stat(parent)
	if err != nil {
		return "", "", publicFSErr(err)
	}
	if !st.IsDir() {
		return "", "", errors.New("路径不是目录")
	}
	name = strings.TrimSpace(name)
	if err := validateName(name); err != nil {
		return "", "", err
	}
	return parent, name, nil
}

func (s *Service) resolvePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("路径无效")
	}
	if strings.ContainsRune(path, 0) {
		return "", errors.New("路径无效")
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("必须使用绝对路径")
	}
	clean := filepath.Clean(path)
	if clean == "." || clean == "" {
		return "", errors.New("路径无效")
	}
	if isUnderTrash(clean) {
		return "", errors.New("请通过回收站管理已删除文件")
	}
	return clean, nil
}

func validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return errors.New("名称无效")
	}
	if strings.ContainsAny(name, `/\`) || strings.ContainsRune(name, 0) {
		return errors.New("名称不能包含路径分隔符")
	}
	if filepath.Base(name) != name {
		return errors.New("名称无效")
	}
	return nil
}

func parentPath(path string) string {
	parent := filepath.Dir(path)
	if parent == path {
		return ""
	}
	// Windows 盘符根如 C:\ 的 Dir 仍是自身
	if vol := filepath.VolumeName(path); vol != "" && filepath.Clean(path) == vol+`\` {
		return ""
	}
	return parent
}

func isRootPath(path string) bool {
	clean := filepath.Clean(path)
	for _, root := range plat.Roots() {
		if filepath.Clean(root) == clean {
			return true
		}
	}
	return clean == string(filepath.Separator)
}

func formatMode(info os.FileInfo) string {
	mode := info.Mode()
	if mode&os.ModeSymlink != 0 {
		return "symlink"
	}
	perm := mode.Perm()
	return fmt.Sprintf("%04o", perm)
}

func isEditableText(raw []byte) bool {
	if len(raw) == 0 {
		return true
	}
	if bytes.IndexByte(raw, 0) >= 0 {
		return false
	}
	return utf8.Valid(raw)
}

func guessLanguage(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".jsx":
		return "javascript"
	case ".json":
		return "json"
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".hpp", ".cxx":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".html", ".htm":
		return "html"
	case ".css", ".less", ".scss":
		return "css"
	case ".xml", ".svg":
		return "xml"
	case ".yml", ".yaml":
		return "yaml"
	case ".md", ".markdown":
		return "markdown"
	case ".sh", ".bash", ".zsh":
		return "shell"
	case ".sql":
		return "sql"
	case ".toml":
		return "ini"
	case ".ini", ".conf", ".cfg", ".env":
		return "ini"
	case ".php":
		return "php"
	case ".rb":
		return "ruby"
	case ".kt":
		return "kotlin"
	case ".swift":
		return "swift"
	case ".vue":
		return "html"
	default:
		return "plaintext"
	}
}

func copyPath(from, to string) error {
	st, err := os.Stat(from)
	if err != nil {
		return publicFSErr(err)
	}
	if st.IsDir() {
		return copyDir(from, to)
	}
	return copyFile(from, to, st.Mode())
}

func copyFile(from, to string, mode os.FileMode) error {
	src, err := os.Open(from)
	if err != nil {
		return publicFSErr(err)
	}
	defer src.Close()
	dst, err := os.OpenFile(to, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode.Perm())
	if err != nil {
		return publicFSErr(err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		_ = os.Remove(to)
		return publicFSErr(err)
	}
	return nil
}

func copyDir(from, to string) error {
	if err := os.MkdirAll(to, 0o755); err != nil {
		return publicFSErr(err)
	}
	entries, err := os.ReadDir(from)
	if err != nil {
		return publicFSErr(err)
	}
	for _, e := range entries {
		src := filepath.Join(from, e.Name())
		dst := filepath.Join(to, e.Name())
		if err := copyPath(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func publicFSErr(err error) error {
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return errors.New("路径不存在")
	}
	if os.IsPermission(err) {
		return errors.New("权限不足")
	}
	if os.IsExist(err) {
		return errors.New("目标已存在")
	}
	msg := err.Error()
	low := strings.ToLower(msg)
	if strings.Contains(low, "permission denied") || strings.Contains(low, "access is denied") {
		return errors.New("权限不足")
	}
	if strings.Contains(low, "no such file") {
		return errors.New("路径不存在")
	}
	if strings.Contains(low, "file exists") {
		return errors.New("目标已存在")
	}
	if strings.Contains(low, "not a directory") {
		return errors.New("路径不是目录")
	}
	if strings.Contains(low, "is a directory") {
		return errors.New("目标是目录")
	}
	return errors.New("文件操作失败")
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
