package files

// ListQuery 目录列表查询。
type ListQuery struct {
	Path     string `form:"path"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// FileItem 一条文件或目录。
type FileItem struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"mod_time"`
	Owner   string `json:"owner"`
}

// ListResult 分页目录列表。
type ListResult struct {
	Path     string     `json:"path"`
	Parent   string     `json:"parent"`
	Items    []FileItem `json:"items"`
	Total    int        `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

// PathRequest 单路径操作。
type PathRequest struct {
	Path string `json:"path"`
}

// NameRequest 在目录下新建或重命名。
type NameRequest struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

// RenameRequest 同目录重命名。
type RenameRequest struct {
	Path    string `json:"path"`
	NewName string `json:"new_name"`
}

// PathsRequest 批量删除。
type PathsRequest struct {
	Paths []string `json:"paths"`
}

// CopyMoveRequest 复制或移动。
type CopyMoveRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// WriteRequest 保存文本文件。
type WriteRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// ReadResult 文本读取结果。
type ReadResult struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	Language string `json:"language"`
}

// TrashQuery 回收站列表查询。
type TrashQuery struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

// TrashItem 回收站条目。
type TrashItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	OriginalPath string `json:"original_path"`
	IsDir        bool   `json:"is_dir"`
	Size         int64  `json:"size"`
	DeletedAt    string `json:"deleted_at"`
}

// TrashListResult 回收站分页。
type TrashListResult struct {
	Items    []TrashItem `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// TrashIDsRequest 按 id 批量操作回收站。
type TrashIDsRequest struct {
	IDs []string `json:"ids"`
}
