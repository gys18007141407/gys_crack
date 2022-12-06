package notifier

import (
	"context"
	"crack_back/src/common"
	"crack_back/src/config"
	"encoding/json"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"path"
	"strconv"
)

// 通知器。通知任务成功或者失败
type Notifier struct {
	Client				*clientv3.Client
	KV					clientv3.KV
	Lease				clientv3.Lease
}

func (This *Notifier) NotifyTaskFinished (task *common.Task) (err error)  {
	var (
		userTask					string
		taskKey						string
		taskValue					[]byte
		op							clientv3.Op
	)
	userTask = path.Join(path.Join(task.TaskType, strconv.Itoa(int(task.UserId)), task.TaskName))
	taskKey = path.Join(config.Cfg.FinishDir, userTask)
	if taskValue, err = json.Marshal(task); err != nil{
		return
	}

	op = clientv3.OpPut(taskKey, string(taskValue), clientv3.WithPrevKV())
	_, err = This.KV.Do(context.TODO(), op)
	return
}

func (This *Notifier) NotifyTaskFailed (task *common.Task) (err error)  {
	var (
		userTask					string
		taskKey						string
		taskValue					[]byte
		op							clientv3.Op
	)
	userTask = path.Join(path.Join(task.TaskType, strconv.Itoa(int(task.UserId)), task.TaskName))
	taskKey = path.Join(config.Cfg.FailDir, userTask)
	if taskValue, err = json.Marshal(task); err != nil{
		return
	}

	op = clientv3.OpPut(taskKey, string(taskValue), clientv3.WithPrevKV())
	_, err = This.KV.Do(context.TODO(), op)
	return
}

// 通知器单例
var(
	Notify 				*Notifier
)

// 初始化单例
func InitNotifier() (err error) {
	if Notify == nil {
		var (
			etcdConfig 			clientv3.Config
			notifier         	Notifier
		)
		etcdConfig = clientv3.Config{
			Endpoints:   config.Cfg.Endpoints,
			DialTimeout: config.Cfg.DialTimeout,
			DialOptions:  []grpc.DialOption{
				grpc.WithBlock(),
			},
		}

		// 建立连接
		if notifier.Client, err = clientv3.New(etcdConfig); err != nil{
			return err
		}

		// 得到KV、Lease、Watcher的子集
		notifier.KV = clientv3.NewKV(notifier.Client)
		notifier.Lease = clientv3.NewLease(notifier.Client)

		// 赋值单例
		Notify = &notifier
	}
	return nil
}
