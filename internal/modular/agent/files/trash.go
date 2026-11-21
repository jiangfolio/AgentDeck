package files

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jiangbyte/agentdeck/internal/config"
)

type trashMeta struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	OriginalPath string `json:"original_path"`
	IsDir        bool   `json:"is_dir"`
	Size         int64  `json:"size"`
	DeletedAt    string `json:"deleted_at"`
}

func trashRoot() string {
	return config.TrashDir()
}

func trashFilesDir() string {
	return filepath.Join(trashRoot(), "files")
}

func trashMetaDir() string {
	return filepath.Join(trashRoot(), "meta")
}

func ensureTrashDirs() error {
	if err := os.MkdirAll(trashFilesDir(), 0o755); err != nil {
		return publicFSErr(err)
	}
	if err := os.MkdirAll(trashMetaDir(), 0o755); err != nil {
		return publicFSErr(err)
	}
	return nil
}

func isUnderTrash(path string) bool {
	root := filepath.Clean(trashRoot())
	clean := filepath.Clean(path)
	if clean == root {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(clean, root+sep)
}

/**
 * TrashList 列出回收站条目。
 */
func (s *Service) TrashList(q TrashQuery) (*TrashListResult, error) {
	// 1. 确保目录存在并读取全部元数据
	if err := ensureTrashDirs(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(trashMetaDir())
	if err != nil {
		return nil, publicFSErr(err)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	items := make([]TrashItem, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		meta, mErr := readTrashMeta(filepath.Join(trashMetaDir(), e.Name()))
		if mErr != nil {
			continue
		}
		if kw != "" &&
			!strings.Contains(strings.ToLower(meta.Name), kw) &&
			!strings.Contains(strings.ToLower(meta.OriginalPath), kw) {
			continue
		}
		items = append(items, TrashItem{
			ID:           meta.ID,
			Name:         meta.Name,
			OriginalPath: meta.OriginalPath,
			IsDir:        meta.IsDir,
			Size:         meta.Size,
			DeletedAt:    meta.DeletedAt,
		})
	}
	// 2. 按删除时间倒序
	sort.Slice(items, func(i, j int) bool {
		return items[i].DeletedAt > items[j].DeletedAt
	})
	// 3. 分页
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
		pageItems = []TrashItem{}
	}
	return &TrashListResult{Items: pageItems, Total: total, Page: page, PageSize: size}, nil
}

/**
 * TrashRestore 还原到原路径；原路径已存在则失败。
 */
func (s *Service) TrashRestore(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(ids) == 0 {
		return errors.New("请选择要还原的项")
	}
	if err := ensureTrashDirs(); err != nil {
		return err
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
			return errors.New("回收站项无效")
		}
		meta, err := readTrashMeta(metaPath(id))
		if err != nil {
			return err
		}
		src := filepath.Join(trashFilesDir(), id)
		dest := meta.OriginalPath
		// 1. 原路径已存在则不覆盖
		if _, err := os.Stat(dest); err == nil {
			return errors.New("原路径已存在，无法还原：" + dest)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return publicFSErr(err)
		}
		// 2. 移回原路径；跨盘失败则复制后删
		if err := os.Rename(src, dest); err != nil {
			if err := copyPath(src, dest); err != nil {
				return err
			}
			_ = os.RemoveAll(src)
		}
		_ = os.Remove(metaPath(id))
	}
	return nil
}

/**
 * TrashPurge 彻底删除回收站项。
 */
func (s *Service) TrashPurge(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(ids) == 0 {
		return errors.New("请选择要删除的项")
	}
	if err := ensureTrashDirs(); err != nil {
		return err
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
			return errors.New("回收站项无效")
		}
		_ = os.RemoveAll(filepath.Join(trashFilesDir(), id))
		_ = os.Remove(metaPath(id))
	}
	return nil
}

/**
 * TrashEmpty 清空回收站。
 */
func (s *Service) TrashEmpty() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ensureTrashDirs(); err != nil {
		return err
	}
	entries, err := os.ReadDir(trashMetaDir())
	if err != nil {
		return publicFSErr(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		_ = os.RemoveAll(filepath.Join(trashFilesDir(), id))
		_ = os.Remove(filepath.Join(trashMetaDir(), e.Name()))
	}
	return nil
}

/**
 * moveToTrash 把文件或目录移入回收站并写元数据。
 */
func moveToTrash(abs string, st os.FileInfo) error {
	// 1. 准备回收站目录与唯一 id
	if err := ensureTrashDirs(); err != nil {
		return err
	}
	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	dest := filepath.Join(trashFilesDir(), id)
	size := st.Size()
	if st.IsDir() {
		size = 0
	}
	meta := trashMeta{
		ID:           id,
		Name:         filepath.Base(abs),
		OriginalPath: abs,
		IsDir:        st.IsDir(),
		Size:         size,
		DeletedAt:    time.Now().Format("2006-01-02 15:04:05"),
	}
	// 2. 先写元数据，再移动内容；移动失败则回滚元数据
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return errors.New("写入回收站失败")
	}
	mp := metaPath(id)
	if err := os.WriteFile(mp, raw, 0o644); err != nil {
		return publicFSErr(err)
	}
	if err := os.Rename(abs, dest); err != nil {
		if err := copyPath(abs, dest); err != nil {
			_ = os.Remove(mp)
			_ = os.RemoveAll(dest)
			return err
		}
		if st.IsDir() {
			if err := os.RemoveAll(abs); err != nil {
				return publicFSErr(err)
			}
		} else if err := os.Remove(abs); err != nil {
			return publicFSErr(err)
		}
	}
	return nil
}

func metaPath(id string) string {
	return filepath.Join(trashMetaDir(), id+".json")
}

func readTrashMeta(path string) (*trashMeta, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, publicFSErr(err)
	}
	var meta trashMeta
	if err := json.Unmarshal(raw, &meta); err != nil || meta.ID == "" {
		return nil, errors.New("回收站记录损坏")
	}
	return &meta, nil
}
