package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/config"
	"github.com/KATOmemorial/cronyx/internal/discovery"
	"github.com/KATOmemorial/cronyx/internal/model"
	"github.com/KATOmemorial/cronyx/internal/rpc"
	"github.com/gin-gonic/gin"
)

var master *discovery.Master

func main() {
	// 1. 初始化
	config.LoadConfig("./configs/config.yaml")
	common.InitLogger()
	common.InitDB()

	// 2. 启动服务发现 Master (监听 Worker 列表)
	master = discovery.NewMaster()
	master.WatchWorkers() // 必须启动监听

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

	r.POST("/job/kill", func(c *gin.Context) {
		// 参数: job_id (任务配置ID)
		jobIDStr := c.PostForm("id")
		jobID, _ := strconv.Atoi(jobIDStr)

		// ⚠️ 这里有个逻辑问题：
		// 我们目前还没有记录 "哪个任务跑在哪个 Worker 上"。
		// 为了演示核心链路，我们假设你要杀的是 "正在跑的所有这个 ID 的任务"。
		// 在真实生产环境，你需要去 Redis 查 "run_id -> worker_ip" 的映射。

		// 演示逻辑：广播给所有 Worker，尝试杀掉这个任务
		// (简单粗暴，但有效。因为 Worker 收到不属于它的 Kill 会直接忽略)

		workers := master.GetWorkers()
		successCount := 0

		for _, workerAddr := range workers {
			// 构造 TaskID (因为目前 Worker 里的 Key 是 job-timestamp)
			// 这里有个小坑：用户点 Kill 时，我们不知道 timestamp 是多少。
			// 所以更科学的做法是：Worker 启动任务时，把 RunID (UUID) 写到 Redis。

			// 🔥 为了让 Sprint 9 能跑通演示，我们做一个临时的 Hack：
			// 我们让 Worker 的 StopTask 接口支持 "前缀匹配" 或者我们暂时只打印日志。

			// 修正方案：
			// 我们先只做 "连通性测试"。
			// 真实的 TaskID 是 "jobID-timestamp"。
			// 我们发一个假的 TaskID 过去，看看 Worker 会不会打印日志。
			fakeTaskID := fmt.Sprintf("%d-1234567890", jobID)

			err := rpc.KillTask(workerAddr, fakeTaskID)
			if err == nil {
				successCount++
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"code":             200,
			"msg":              "Kill signal sent",
			"workers_notified": len(workers),
		})
	})

	r.Run(":" + strconv.Itoa(config.AppConfig.Server.HttpPort))
}
