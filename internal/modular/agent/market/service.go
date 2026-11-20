package market

import (
	_ "embed"
	"encoding/json"
	"os/exec"
	"sort"
	"strings"
)

//go:embed catalog.json
var catalogJSON []byte

// Service 应用市场目录（仅展示，不安装）。
type Service struct {
	apps []App
}

func NewService() *Service {
	var apps []App
	_ = json.Unmarshal(catalogJSON, &apps)
	return &Service{apps: apps}
}

/**
 * Catalog 按页签、分类、关键字过滤静态目录，并用 PATH 判断是否已安装。
 */
func (s *Service) Catalog(q CatalogQuery) *CatalogResult {
	// 1. 校正分页，避免空页或超大 page_size
	page := q.Page
	if page < 1 {
		page = 1
	}
	size := q.PageSize
	if size < 1 {
		size = 20
	}
	if size > 60 {
		size = 60
	}
	tab := strings.ToLower(strings.TrimSpace(q.Tab))
	if tab == "" {
		tab = "all"
	}
	keyword := strings.ToLower(strings.TrimSpace(q.Keyword))
	category := strings.TrimSpace(q.Category)

	// 2. 暂时只保留本机 PATH 里能找到的应用，未安装的目录项先不展示
	catSet := map[string]struct{}{}
	matched := make([]App, 0, len(s.apps))
	for _, app := range s.apps {
		installed := app.Bin != "" && lookPath(app.Bin)
		if !installed || tab == "upgradable" {
			continue
		}
		if app.Category != "" {
			catSet[app.Category] = struct{}{}
		}
		if category != "" && category != "全部" && app.Category != category {
			continue
		}
		if keyword != "" &&
			!strings.Contains(strings.ToLower(app.Name), keyword) &&
			!strings.Contains(strings.ToLower(app.Summary), keyword) {
			continue
		}
		app.Installed = installed
		matched = append(matched, app)
	}

	// 3. 分类稳定排序后分页返回
	categories := make([]string, 0, len(catSet))
	for c := range catSet {
		categories = append(categories, c)
	}
	sort.Strings(categories)
	total := len(matched)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	items := matched[start:end]
	if items == nil {
		items = []App{}
	}
	return &CatalogResult{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   size,
		Categories: categories,
	}
}

func lookPath(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
