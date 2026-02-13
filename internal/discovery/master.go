package discovery

import (
	"context"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/config"
)

// Master 服务发现客户端 (API/Scheduler 用)
type Master struct {
	cli       *clientv3.Client
	workerMap map[string]string // 本地缓存: IP -> Info
	lock      sync.Mutex
}

func NewMaster() *Master {
	// 初始化 Etcd 连接
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   config.AppConfig.Etcd.Endpoints,
		DialTimeout: time.Duration(config.AppConfig.Etcd.DialTimeout) * time.Second,
	})
	if err != nil {
		common.Log.Fatal("Failed to connect to Etcd", zap.Error(err))
	}

	return &Master{
		cli:       cli,
		workerMap: make(map[string]string),
	}
}

// WatchWorkers 监听 /cronyx/worker/ 目录
func (m *Master) WatchWorkers() {
	// 1. 先 Get 一次现有的所有 Worker
	resp, err := m.cli.Get(context.Background(), "/cronyx/worker/", clientv3.WithPrefix())
	if err != nil {
		common.Log.Error("Failed to get existing workers", zap.Error(err))
	} else {
		for _, kv := range resp.Kvs {
			m.addWorker(string(kv.Key), string(kv.Value))
		}
	}

	// 2. 启动 Watch 协程
	go func() {
		// 监听 /cronyx/worker/ 后续的变化
		watchChan := m.cli.Watch(context.Background(), "/cronyx/worker/", clientv3.WithPrefix())

		for resp := range watchChan {
			for _, event := range resp.Events {
				switch event.Type {
				case mvccpb.PUT: // 新增或修改
					m.addWorker(string(event.Kv.Key), string(event.Kv.Value))
				case mvccpb.DELETE: // 删除 (节点下线/过期)
					m.delWorker(string(event.Kv.Key))
				}
			}
		}
	}()
}

// GetWorkers 获取当前所有活着的 Worker
func (m *Master) GetWorkers() map[string]string {
	m.lock.Lock()
	defer m.lock.Unlock()
	// 返回副本，防止并发读写冲突
	copyMap := make(map[string]string)
	for k, v := range m.workerMap {
		copyMap[k] = v
	}
	return copyMap
}

// 内部方法：添加 Worker
func (m *Master) addWorker(key, value string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// key: /cronyx/worker/192.168.1.5:9999 -> ID: 192.168.1.5:9999
	// 这里简单处理，直接用 key 做 ID，或者你可以解析一下 IP
	m.workerMap[key] = value
	common.Log.Info("Worker Added", zap.String("node", key))
}

// 内部方法：删除 Worker
func (m *Master) delWorker(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.workerMap, key)
	common.Log.Warn("Worker Removed", zap.String("node", key))
}
