package common

type TaskEvent struct {
	TaskID    string `json:"task_id"`
	Command   string `json:"command"`
	Timestamp int64  `json:"timestamp"`
}
