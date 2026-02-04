package model

import "gorm.io/gorm"

// JobLog 任务执行日志
type JobLog struct {
	gorm.Model

	// 关联 JobInfo (方便联表查询)
	JobID uint `gorm:"not null;index;comment:任务ID" json:"job_id"`

	// 执行信息
	Command string `gorm:"type:text;comment:执行命令" json:"command"`
	Output  string `gorm:"type:mediumtext;comment:执行输出(标准输出+错误)" json:"output"`
	Error   string `gorm:"type:text;comment:错误信息" json:"error"`

	// 性能指标
	PlanTime  int64 `gorm:"comment:计划执行时间" json:"plan_time"`
	RealTime  int64 `gorm:"comment:实际调度时间" json:"real_time"`
	StartTime int64 `gorm:"comment:开始执行时间" json:"start_time"`
	EndTime   int64 `gorm:"comment:执行结束时间" json:"end_time"`

	// 结果状态
	Status int `gorm:"default:0;comment:0:失败 1:成功" json:"status"`
}
