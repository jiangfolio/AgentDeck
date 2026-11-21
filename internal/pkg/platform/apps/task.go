package apps

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Task 异步导入任务状态。
type Task struct {
	ID          string  `json:"id"`
	AppID       string  `json:"app_id"`
	Version     string  `json:"version"`
	Phase       string  `json:"phase"`
	Progress    float64 `json:"progress"`
	Message     string  `json:"message"`
	Error       string  `json:"error,omitempty"`
	InstallPath string  `json:"install_path,omitempty"`
	Done        bool    `json:"done"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
}

// TaskStore 内存任务表（面板单机足够）。
type TaskStore struct {
	mu    sync.Mutex
	items map[string]*Task
}

func NewTaskStore() *TaskStore {
	return &TaskStore{items: map[string]*Task{}}
}

/**
 * Create 新建任务并返回副本。
 */
func (s *TaskStore) Create(appID, version string) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().Unix()
	t := &Task{
		ID:        uuid.NewString(),
		AppID:     appID,
		Version:   version,
		Phase:     "queued",
		Progress:  0,
		Message:   "等待处理",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.items[t.ID] = t
	cp := *t
	return &cp
}

/**
 * Update 更新任务进度；appID/version 非空时同步写入。
 */
func (s *TaskStore) Update(id, phase, message string, progress float64, errMsg string, done bool) {
	s.UpdateMeta(id, "", "", phase, message, progress, errMsg, done)
}

/**
 * UpdateMeta 更新任务，可同时纠正解析出的 app/version。
 */
func (s *TaskStore) UpdateMeta(id, appID, version, phase, message string, progress float64, errMsg string, done bool) {
	s.Finish(id, appID, version, "", phase, message, progress, errMsg, done)
}

/**
 * Finish 完成或推进任务，可写入安装路径供前端跳转文件管理。
 */
func (s *TaskStore) Finish(id, appID, version, installPath, phase, message string, progress float64, errMsg string, done bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[id]
	if !ok {
		return
	}
	if appID != "" {
		t.AppID = appID
	}
	if version != "" {
		t.Version = version
	}
	if installPath != "" {
		t.InstallPath = installPath
	}
	t.Phase = phase
	t.Message = message
	t.Progress = progress
	t.Error = errMsg
	t.Done = done
	t.UpdatedAt = time.Now().Unix()
}

/**
 * Get 返回任务副本。
 */
func (s *TaskStore) Get(id string) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[id]
	if !ok {
		return nil
	}
	cp := *t
	return &cp
}
