package discovery

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/config"
)

// ServiceRegister 服务注册器 (Worker 用)
type ServiceRegister struct {
	cli     *clientv3.Client // Etcd 客户端
	leaseID clientv3.LeaseID // 租约 ID
	key     string           // 注册的 Key
	val     string           // 注册的 Value
}

// NewServiceRegister 创建注册器
func NewServiceRegister() *ServiceRegister {
	// 1. 初始化 Etcd 客户端
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   config.AppConfig.Etcd.Endpoints,
		DialTimeout: time.Duration(config.AppConfig.Etcd.DialTimeout) * time.Second,
	})
	if err != nil {
		common.Log.Fatal("Failed to connect to Etcd", zap.Error(err))
	}

	return &ServiceRegister{
		cli: cli,
	}
}

// Register 注册服务 (带租约 + 自动续租)
func (s *ServiceRegister) Register(key, value string, ttl int64) error {
	s.key = key
	s.val = value

	// 1. 创建租约 (Lease)
	ctx := context.TODO()
	resp, err := s.cli.Grant(ctx, ttl)
	if err != nil {
		return err
	}
	s.leaseID = resp.ID

	// 2. 写入 KV (绑定租约)
	// 当租约过期时，这个 KV 会自动被 Etcd 删除 —— 这就是“服务下线”的原理
	if _, err := s.cli.Put(ctx, key, value, clientv3.WithLease(s.leaseID)); err != nil {
		return err
	}

	// 3. 自动续租 (KeepAlive)
	// 启动一个协程，每秒告诉 Etcd "我还活着"
	keepAliveChan, err := s.cli.KeepAlive(ctx, s.leaseID)
	if err != nil {
		return err
	}

	// 异步处理续租应答
	go func() {
		// 使用 for range 遍历通道
		// 1. 只要 channel 有数据，循环就会执行（续租成功）
		// 2. 一旦 channel 关闭，循环会自动结束
		for range keepAliveChan {
			// 续租成功，静默处理
			// 如果想看日志，可以在这里加：common.Log.Debug("Lease renewed")
		}

		// 循环退出说明 channel 已经关闭了
		common.Log.Error("Etcd KeepAlive channel closed, leasing revoked")
	}()

	common.Log.Info("Service Registered to Etcd", zap.String("key", key), zap.Int64("ttl", ttl))
	return nil
}

// Close 注销服务
func (s *ServiceRegister) Close() {
	// 撤销租约，立即删除 Key
	if _, err := s.cli.Revoke(context.Background(), s.leaseID); err != nil {
		common.Log.Error("Failed to revoke lease", zap.Error(err))
	}
	common.Log.Info("Service Unregistered", zap.String("key", s.key))
}
