package service

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"github.com/KATOmemorial/cronyx/internal/biz"
	"github.com/KATOmemorial/cronyx/internal/model"
	"github.com/KATOmemorial/cronyx/pkg/response" // 确保你之前创建了这个包
)

// ProviderSet 导出
var ProviderSet = wire.NewSet(NewJobService)

type JobService struct {
	uc *biz.JobUseCase
}

// NewJobService 注入 biz.JobUseCase
func NewJobService(uc *biz.JobUseCase) *JobService {
	return &JobService{uc: uc}
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

// KillHandler 强杀 (暂时留空或简单的逻辑，后续补全)
func (s *JobService) KillHandler(c *gin.Context) {
	// ... 逻辑后续补充
	response.Success(c, nil)
}
