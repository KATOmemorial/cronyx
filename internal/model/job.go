package model

import (
	"gorm.io/gorm"
)

const (
	JobTypeShell = 1
	JobTypeHttp  = 2
)

type JobInfo struct {
	gorm.Model

	Name        string `gorm:"type:varchar(100);not null;comment:任务名称" json:"name"`
	Description string `gorm:"type:varchar(255);comment:任务描述" json:"description"`

	CronExpr string `gorm:"type:varchar(50);not null;comment:Cron表达式" json:"cron_expr"`
	Command  string `gorm:"type:text;not null;comment:执行命令或URL" json:"command"`
	JobType  int    `gorm:"default:1;comment:任务类型 1:Shell 2:HTTP" json:"job_type"`

	Status int `gorm:"default:0;comment:状态 0:停止 1:启动" json:"status"`

	NextTime int64 `gorm:"index;comment:下次执行时间戳" json:"next_time"`
}
