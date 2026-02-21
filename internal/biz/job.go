package biz

import (
	"context"
	"time"

	"github.com/google/wire"
	"go.uber.org/zap"

	"github.com/KATOmemorial/cronyx/internal/model"
)

// ProviderSet 导出给 Wire
var ProviderSet = wire.NewSet(NewJobUseCase)

// JobRepo 接口定义 (由 data 层实现)
// 这样做实现了依赖倒置：biz 层不依赖 data 层，而是 data 层依赖 biz 层的接口定义
type JobRepo interface {
	Create(ctx context.Context, job *model.JobInfo) error
	Update(ctx context.Context, job *model.JobInfo) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.JobInfo, error)
	List(ctx context.Context, page, size int) ([]*model.JobInfo, int64, error)
	ListLogs(ctx context.Context, jobID uint, limit int) ([]*model.JobLog, error)
	CreateLog(ctx context.Context, log *model.JobLog) error
}

// JobUseCase 业务逻辑用例
type JobUseCase struct {
	repo JobRepo
	log  *zap.Logger
}

// NewJobUseCase 构造函数
func NewJobUseCase(repo JobRepo, logger *zap.Logger) *JobUseCase {
	return &JobUseCase{
		repo: repo,
		log:  logger,
	}
}

// Create 创建任务
func (uc *JobUseCase) Create(ctx context.Context, job *model.JobInfo) error {
	// 业务逻辑：设置初始下次执行时间为当前时间 (立即调度或按 Cron 计算，这里简化为立即)
	if job.NextTime == 0 {
		job.NextTime = time.Now().Unix()
	}
	// 业务逻辑：默认为停止状态
	// job.Status = 0

	return uc.repo.Create(ctx, job)
}

// Update 更新任务
func (uc *JobUseCase) Update(ctx context.Context, job *model.JobInfo) error {
	// 可以在这里增加 Cron 表达式校验逻辑
	return uc.repo.Update(ctx, job)
}

// Delete 删除任务
func (uc *JobUseCase) Delete(ctx context.Context, id uint) error {
	return uc.repo.Delete(ctx, id)
}

// List 获取任务列表
func (uc *JobUseCase) List(ctx context.Context, page, size int) (map[string]interface{}, error) {
	jobs, total, err := uc.repo.List(ctx, page, size)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"list":  jobs,
		"total": total,
		"page":  page,
		"size":  size,
	}, nil
}

// GetLogs 获取日志
func (uc *JobUseCase) GetLogs(ctx context.Context, jobID uint) ([]*model.JobLog, error) {
	// 默认只查最近 20 条
	return uc.repo.ListLogs(ctx, jobID, 20)
}
