package service

import (
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"go.uber.org/zap"

	"github.com/KATOmemorial/cronyx/internal/biz"
	"github.com/KATOmemorial/cronyx/internal/discovery"
	"github.com/KATOmemorial/cronyx/internal/model"
	"github.com/KATOmemorial/cronyx/internal/rpc"
	"github.com/KATOmemorial/cronyx/pkg/response"
)

// ProviderSet 导出
var ProviderSet = wire.NewSet(NewJobService)

type JobService struct {
	uc     *biz.JobUseCase
	master *discovery.Master
	log    *zap.Logger
}

// NewJobService 注入依赖
func NewJobService(uc *biz.JobUseCase, master *discovery.Master, logger *zap.Logger) *JobService {
	return &JobService{
		uc:     uc,
		master: master,
		log:    logger,
	}
}

// CreateHandler 创建任务
func (s *JobService) CreateHandler(c *gin.Context) {
	var job model.JobInfo
	if err := c.ShouldBindJSON(&job); err != nil {
		response.Error(c, 400, "Invalid params: "+err.Error())
		return
	}
	if err := s.uc.Create(c.Request.Context(), &job); err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	response.Success(c, job)
}

// ListHandler 列表
func (s *JobService) ListHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	data, err := s.uc.List(c.Request.Context(), page, size)
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}
	response.Success(c, data)
}

// KillReq 强杀请求参数
type KillReq struct {
	TaskID string `json:"task_id" binding:"required"`
}

// KillHandler 强杀任务
func (s *JobService) KillHandler(c *gin.Context) {
	var req KillReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, "Invalid task_id")
		return
	}

	// 1. 获取所有活着的 Worker 节点
	workers := s.master.GetWorkers()
	if len(workers) == 0 {
		response.Error(c, 500, "No active workers found in cluster")
		return
	}

	s.log.Info("Broadcasting kill signal", zap.String("task_id", req.TaskID), zap.Int("worker_count", len(workers)))

	// 2. 并发广播强杀指令 (集群广播法)
	var wg sync.WaitGroup
	killSuccess := false // 只要有一个 worker 杀成功了，我们就认为成功了
	var mu sync.Mutex

	for _, addr := range workers {
		wg.Add(1)
		go func(workerAddr string) {
			defer wg.Done()
			err := rpc.KillTask(workerAddr, req.TaskID, s.log)
			if err == nil {
				// 没有报错，说明正好是这个 worker 运行了该任务并成功杀掉
				mu.Lock()
				killSuccess = true
				mu.Unlock()
			}
		}(addr) // 传参防止闭包问题
	}

	wg.Wait()

	if killSuccess {
		response.Success(c, "Task killed successfully across cluster")
	} else {
		// 都没成功，可能任务已经执行完了，或者根本不存在
		response.Success(c, "Task not found running on any worker (might be finished already)")
	}
}

// LogHandler 获取任务的执行日志
func (s *JobService) LogHandler(c *gin.Context) {
	// 1. 从 URL 路径中获取 id 参数 (例如 /job/1/logs)
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Error(c, 400, "Invalid job ID")
		return
	}

	// 2. 调用业务层获取日志 (默认拉取最近 20 条)
	logs, err := s.uc.GetLogs(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, 500, err.Error())
		return
	}

	// 3. 返回给前端
	response.Success(c, logs)
}
