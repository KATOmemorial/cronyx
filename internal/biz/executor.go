package biz

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Executor 负责管理任务的执行和强杀
type Executor struct {
	log      *zap.Logger
	taskMap  map[string]context.CancelFunc // 运行中的任务: TaskID -> CancelFunc
	taskLock sync.Mutex
}

func NewExecutor(logger *zap.Logger) *Executor {
	return &Executor{
		log:     logger,
		taskMap: make(map[string]context.CancelFunc),
	}
}

// StartExecution 启动一个 Shell 任务
// command: "sleep 10"
// taskID: "101-17000000"
func (e *Executor) StartExecution(ctx context.Context, taskID, command string) (string, error) {
	// 1. 创建可取消的 Context
	runCtx, cancel := context.WithCancel(ctx)

	// 2. 登记任务
	e.taskLock.Lock()
	e.taskMap[taskID] = cancel
	e.taskLock.Unlock()

	// 3. 执行命令
	startTime := time.Now()
	cmd := exec.CommandContext(runCtx, "/bin/sh", "-c", command)
	output, err := cmd.CombinedOutput() // 阻塞直到执行完成或被 Kill

	// 4. 执行结束，注销任务
	e.taskLock.Lock()
	delete(e.taskMap, taskID)
	e.taskLock.Unlock()

	cost := time.Since(startTime)
	e.log.Info("Job finished",
		zap.String("task_id", taskID),
		zap.Int64("cost_ms", cost.Milliseconds()),
	)

	return string(output), err
}

// KillTask 强杀任务
// targetID: 支持前缀匹配，例如 "101" 会杀掉 "101-17000"
func (e *Executor) KillTask(targetID string) int {
	e.taskLock.Lock()
	defer e.taskLock.Unlock()

	count := 0
	for taskID, cancel := range e.taskMap {
		if strings.HasPrefix(taskID, targetID) {
			cancel() // 触发 CommandContext 的 Kill
			delete(e.taskMap, taskID)
			count++
			e.log.Warn("💀 Task killed by user", zap.String("task_id", taskID))
		}
	}
	return count
}
