package main

import (
	"net/http"
	"time"

	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/model"
	"github.com/gin-gonic/gin"
)

func main() {
	common.InitDB()

	r := gin.Default()

	r.POST("/job", func(c *gin.Context) {
		var job model.JobInfo

		if err := c.ShouldBindJSON(&job); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if job.NextTime == 0 {
			job.NextTime = time.Now().Unix()
		}

		if err := common.DB.Create(&job).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Job created", "data": job})
	})

	r.GET("jobs", func(c *gin.Context) {
		var jobs []model.JobInfo
		common.DB.Find(&jobs)
		c.JSON(http.StatusOK, gin.H{"data": jobs})
	})

	r.GET("/job/:id/logs", func(c *gin.Context) {
		jobID := c.Param("id")
		var logs []model.JobLog

		if err := common.DB.Where("job_id = ?", jobID).Order("id desc").Limit(20).Find(&logs).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch logs"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": logs})
	})

	r.Run(":8080")
}
