package discovery

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/google/wire"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"

	"github.com/KATOmemorial/cronyx/internal/config"
)

// ElectionProviderSet 导出给 Wire
var ElectionProviderSet = wire.NewSet(NewElection)

// Election 封装 Etcd 选主逻辑
type Election struct {
	cli      *clientv3.Client
	log      *zap.Logger
	isLeader int32 // 使用原子操作保证并发安全 (0: false, 1: true)
}

// NewElection 构造函数
func NewElection(conf *config.Config, logger *zap.Logger) (*Election, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   conf.Etcd.Endpoints,
		DialTimeout: time.Duration(conf.Etcd.DialTimeout) * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &Election{
		cli:      cli,
		log:      logger,
		isLeader: 0,
	}, nil
}

// IsLeader 线程安全地查询当前是否是 Leader
func (e *Election) IsLeader() bool {
	return atomic.LoadInt32(&e.isLeader) == 1
}

// Campaign 开始后台竞选 (非阻塞)
func (e *Election) Campaign(ctx context.Context, electionKey, nodeVal string) {
	go func() {
		for {
			// 检查是否收到退出信号
			select {
			case <-ctx.Done():
				return
			default:
			}

			// 1. 创建租约 Session (10秒TTL，Etcd会自动帮我们续租)
			session, err := concurrency.NewSession(e.cli, concurrency.WithTTL(10))
			if err != nil {
				e.log.Error("Failed to create etcd session", zap.Error(err))
				time.Sleep(3 * time.Second)
				continue
			}

			election := concurrency.NewElection(session, electionKey)
			e.log.Info("🙋‍♂️ Node starting campaign...", zap.String("key", electionKey))

			// 2. 阻塞竞选 (只有当选 Leader 才会往下走，否则一直卡在这里等)
			if err := election.Campaign(ctx, nodeVal); err != nil {
				session.Close()
				time.Sleep(3 * time.Second)
				continue
			}

			// 3. 竞选成功，我就是 Leader！👑
			atomic.StoreInt32(&e.isLeader, 1)
			e.log.Info("👑 I am the LEADER now!", zap.String("val", nodeVal))

			// 4. 持续监听，如果网络断开导致 Session 失效，需要退位让贤
			select {
			case <-session.Done():
				e.log.Warn("⚠️ Session expired, stepping down as leader")
				atomic.StoreInt32(&e.isLeader, 0)
			case <-ctx.Done():
				e.log.Info("🛑 Context canceled, stepping down")
				election.Resign(context.Background()) // 主动退位
				atomic.StoreInt32(&e.isLeader, 0)
				session.Close()
				return
			}
		}
	}()
}

// Close 关闭连接
func (e *Election) Close() {
	if e.cli != nil {
		e.cli.Close()
	}
}
