package market

// CatalogQuery 应用目录筛选。
type CatalogQuery struct {
	Tab       string `form:"tab"`
	Category  string `form:"category"`
	Keyword   string `form:"keyword"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	LocalOnly bool   `form:"local_only"`
}

// App 目录中的一条应用。
type App struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	Category  string `json:"category"`
	Bin       string `json:"bin"`
	Installed bool   `json:"installed"`
}

// CatalogResult 分页后的目录。
type CatalogResult struct {
	Items      []App    `json:"items"`
	Total      int      `json:"total"`
	Page       int      `json:"page"`
	PageSize   int      `json:"page_size"`
	Categories []string `json:"categories"`
}
