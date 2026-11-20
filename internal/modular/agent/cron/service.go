package cron

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jiangbyte/agentdeck/internal/config"
	platcron "github.com/jiangbyte/agentdeck/internal/pkg/platform/cron"
	robfig "github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Service 定时任务业务。
type Service struct {
	repo *Repo
	sch  platcron.Scheduler
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo, sch: platcron.New()}
}

/**
 * Reconcile 启动时以 DB 为准同步系统调度器。
 */
func (s *Service) Reconcile() error {
	jobs, err := s.repo.ListAll()
	if err != nil {
		return err
	}
	specs := make([]platcron.JobSpec, 0, len(jobs))
	for _, j := range jobs {
		specs = append(specs, toSpec(j))
	}
	return s.sch.Reconcile(specs)
}

/**
 * List 分页列出任务并附带下次运行时间。
 */
func (s *Service) List(q ListQuery) (*ListResult, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	items, total, err := s.repo.List(q.Page, q.PageSize, q.Enabled)
	if err != nil {
		return nil, err
	}
	views := make([]JobView, 0, len(items))
	for _, j := range items {
		views = append(views, JobView{Job: j, NextRunAt: nextRunUnix(j.Expr, j.Enabled)})
	}
	return &ListResult{Items: views, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

/**
 * Save 创建或更新任务，事务外同步 OS 调度器。
 */
func (s *Service) Save(req SaveRequest) (*JobView, error) {
	// 1. 校验五段 cron
	if err := validateExpr(req.Expr); err != nil {
		return nil, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	var job *Job
	var err error
	if strings.TrimSpace(req.ID) == "" {
		// 2. 新建
		job = &Job{
			ID:      uuid.NewString(),
			Name:    strings.TrimSpace(req.Name),
			Expr:    strings.TrimSpace(req.Expr),
			Command: req.Command,
			WorkDir: req.WorkDir,
			Enabled: enabled,
		}
		if err = s.repo.Create(job); err != nil {
			return nil, fmt.Errorf("保存失败")
		}
	} else {
		// 3. 更新：先查再改
		job, err = s.repo.FindByID(req.ID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, fmt.Errorf("任务不存在")
			}
			return nil, err
		}
		job.Name = strings.TrimSpace(req.Name)
		job.Expr = strings.TrimSpace(req.Expr)
		job.Command = req.Command
		job.WorkDir = req.WorkDir
		job.Enabled = enabled
		if err = s.repo.Update(job); err != nil {
			return nil, fmt.Errorf("更新失败")
		}
	}

	// 4. 事务外同步系统调度器
	if err = s.sch.Apply(toSpec(*job)); err != nil {
		return nil, fmt.Errorf("同步系统调度器失败: %v", err)
	}
	return &JobView{Job: *job, NextRunAt: nextRunUnix(job.Expr, job.Enabled)}, nil
}

/**
 * Remove 先从 OS 移除再删库。
 */
func (s *Service) Remove(id string) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("任务不存在")
		}
		return err
	}
	if err := s.sch.Remove(id); err != nil {
		return fmt.Errorf("移除系统任务失败: %v", err)
	}
	return s.repo.Delete(id)
}

/**
 * SetEnabled 启停并同步 OS。
 */
func (s *Service) SetEnabled(id string, enabled bool) error {
	job, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("任务不存在")
		}
		return err
	}
	job.Enabled = enabled
	if err := s.repo.Update(job); err != nil {
		return fmt.Errorf("更新失败")
	}
	if err := s.sch.Apply(toSpec(*job)); err != nil {
		return fmt.Errorf("同步系统调度器失败: %v", err)
	}
	return nil
}

/**
 * Runs 读取最近运行记录（最多 50 条，新→旧）。
 */
func (s *Service) Runs(id string) ([]RunRecord, error) {
	if _, err := s.repo.FindByID(id); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, err
	}
	path := filepath.Join(config.CronLogDir(), id, "runs.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []RunRecord{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var all []RunRecord
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var rec RunRecord
		if json.Unmarshal([]byte(line), &rec) == nil {
			all = append(all, rec)
		}
	}
	if len(all) > 50 {
		all = all[len(all)-50:]
	}
	// 新→旧
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	return all, nil
}

func toSpec(j Job) platcron.JobSpec {
	return platcron.JobSpec{
		ID: j.ID, Name: j.Name, Expr: j.Expr,
		Command: j.Command, WorkDir: j.WorkDir, Enabled: j.Enabled,
	}
}

func validateExpr(expr string) error {
	parser := robfig.NewParser(robfig.Minute | robfig.Hour | robfig.Dom | robfig.Month | robfig.Dow)
	_, err := parser.Parse(strings.TrimSpace(expr))
	if err != nil {
		return fmt.Errorf("cron 表达式无效，需五段格式")
	}
	return nil
}

func nextRunUnix(expr string, enabled bool) int64 {
	if !enabled {
		return 0
	}
	parser := robfig.NewParser(robfig.Minute | robfig.Hour | robfig.Dom | robfig.Month | robfig.Dow)
	sched, err := parser.Parse(strings.TrimSpace(expr))
	if err != nil {
		return 0
	}
	return sched.Next(time.Now()).Unix()
}
