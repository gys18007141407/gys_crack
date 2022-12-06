package alerter

import (
	"context"
	"crack_front/src/common"
	"crack_front/src/config"
	"crack_front/src/master/logger"
	"encoding/json"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"strings"
)

// 警报器 由选举器决定该master是否启动警报器
// master是无状态的(状态/数据在etcd中)，故而我们可以同时开启多个master来服务
// master选主，主master监听/cron/warn/目录，任务失败时示警。worker向该目录put任务警告信息，主master[防止重复通知]负责通知给负责人
// etcd写入性能差，可能来不及，改为消息队列

type Alerter struct {
	client 				*clientv3.Client
	lease				clientv3.Lease
	kv 					clientv3.KV
	watcher 			clientv3.Watcher
	warnDir				string

	warnMessageChan		chan *common.WarnMessage // 预警信息管道
}

// 持续监听警报任务，直到警报器上下文取消
func (This *Alerter) Start(ctx context.Context)  {
	var (
		err 				error
		op 					clientv3.Op
		opResp				clientv3.OpResponse
		watchRevision		int64
		watchChan 			clientv3.WatchChan
		watchResp			clientv3.WatchResponse
		watchEvent 			*clientv3.Event

		warnMessage 		*common.WarnMessage
	)

	for {
		// 获取etcd中/cron/warn/目录下的所有警报任务，以及当前集群的revision
		// TODO：当Leader选举较慢的时候有可能丢掉某些警报(worker放置警告时的租约时长默认为1s)
		op = clientv3.OpGet(This.warnDir, clientv3.WithPrefix())
		if opResp, err = This.kv.Do(ctx, op); err != nil {
			goto RETRY
		}
		watchRevision = opResp.Get().Header.Revision+1
		watchChan = This.watcher.Watch(ctx, This.warnDir, clientv3.WithPrefix(), clientv3.WithRev(watchRevision))
		// 遍历watch的事件, 直到watchChan关闭
		for watchResp = range watchChan{
			for _, watchEvent = range watchResp.Events{
				switch watchEvent.Type {
				case clientv3.EventTypePut:		// 修改事件意味着worker新增了一个警报任务
					// 反序列化value警报信息
					warnMessage = &common.WarnMessage{}
					if err = json.Unmarshal(watchEvent.Kv.Value, warnMessage); err != nil{
						warnMessage.TaskName = strings.TrimPrefix(string(watchEvent.Kv.Key), This.warnDir)
						warnMessage.Message = "预警信息反序列化失败了"
					}
					// 通知loop协程
					This.warnMessageChan <- warnMessage
				case clientv3.EventTypeDelete:	// 不在乎删除事件，这是由于worker放置的key的租约到期了
				}
			}
		}
		// watchChan关闭
	RETRY:
		select {
		case <-ctx.Done():  // 不再是Leader了, 退出start
			return
		default:			// 其他错误，继续循环
		}
	}
}

// 发送所有的警报信息
func (This *Alerter) loop()  {
	var(
		warnMessage 		*common.WarnMessage
	)
	for{
		select {
		case warnMessage = <-This.warnMessageChan:
			// 发送给相关管理人员
			logger.Logger.WarnLog(warnMessage)
		}
	}
}

// 警报器单例
var (
	Alert			*Alerter
)

// 初始化警报器
func InitAlerter() (err error) {
	var(
		etcdCfg		clientv3.Config
		client		*clientv3.Client
	)
	etcdCfg = clientv3.Config{
		Endpoints: config.Cfg.Endpoints,
		DialTimeout: config.Cfg.DialTimeout,
		DialOptions:   []grpc.DialOption{
			grpc.WithBlock(),
		},
	}
	if client, err = clientv3.New(etcdCfg); err != nil{
		return
	}

	// 初始化单例
	Alert = &Alerter{
		client:          client,
		lease:           clientv3.NewLease(client),
		kv:              clientv3.NewKV(client),
		watcher:         clientv3.NewWatcher(client),
		warnDir:         config.Cfg.WarnDir,
		warnMessageChan: make(chan *common.WarnMessage, 512),
	}

	// 发送预警信息
	go Alert.loop()

	return
}