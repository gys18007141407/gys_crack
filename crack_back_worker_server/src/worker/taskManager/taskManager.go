package taskManager

import (
	"context"
	"crack_back/src/common"
	"crack_back/src/config"
	"crack_back/src/worker/lock"
	"crack_back/src/worker/logger"
	"crack_back/src/worker/scheduler"
	"encoding/json"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"strings"
)

// 一个etcd客户端，用来管理任务
type TaskManager struct {
	Client 		*clientv3.Client
	KV 			clientv3.KV
	Lease 		clientv3.Lease
	Watcher  	clientv3.Watcher
}

// 该客户端绑定的方法
func (This *TaskManager) WatchTasks() (err error) {
	var (
		Op 					clientv3.Op
		OpResp 				clientv3.OpResponse

		kvPair 				*mvccpb.KeyValue
		temp 				*common.Task

		taskEvent			*common.TaskEvent
	)
	// 获取etcd中/crack/tasks/目录下的所有任务，以及当前集群的revision
	Op = clientv3.OpGet(config.Cfg.TaskDir, clientv3.WithPrefix())
	if OpResp, err = This.KV.Do(context.TODO(), Op); err != nil{
		return
	}

	// 遍历Response, 并进行反序列化, 对etcd中的任务进行全量同步(不全量同步，防止同一个任务执行两次)
	for _, kvPair = range OpResp.Get().Kvs{
		temp = &common.Task{}
		if err = json.Unmarshal(kvPair.Value, temp); err != nil {
			logger.Logger.InfoLog("WatchTasks反序列化错误...已丢弃该错误:", err.Error())
		}else{
			// 获得任务
			taskEvent = &common.TaskEvent{
				CurEvent: common.EventSave,
				CurTask:  temp,
			}
			// 给调度器的事件队列推送一个新事件
			scheduler.Sched.PushTaskEvent(taskEvent)
		}
	}

	// 一个协程持续监听该目录的revision之后版本的变化事件
	go func() {
		var(
			watchStartRevision	int64
			watchRespChan		clientv3.WatchChan
			watchResp			clientv3.WatchResponse
			watchEvent			*clientv3.Event
			task 				*common.Task
			taskBaseName		string
		)
		watchStartRevision = OpResp.Get().Header.Revision + 1
		watchRespChan = This.Watcher.Watch(context.TODO(), config.Cfg.TaskDir, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix())

		// 遍历watch的应答
		for watchResp = range watchRespChan{
			// 遍历应答的事件
			for _, watchEvent = range watchResp.Events{
				switch watchEvent.Type {
				// 保存任务
				case clientv3.EventTypePut:
					// 反序列化任务并推给scheduler调度器一个更新事件
					task = &common.Task{}
					if err = json.Unmarshal(watchEvent.Kv.Value, task); err != nil{
						logger.Logger.InfoLog("WatchEvent反序列化错误...已丢弃该错误:", err.Error())
					}
					// 推给scheduler调度器一个更新事件
					taskEvent = &common.TaskEvent{
						CurEvent: common.EventSave,
						CurTask:  task,
					}
					break

				// 删除任务
				case clientv3.EventTypeDelete:
					// 获取任务名(不包括目录名),推给scheduler调度器一个删除事件
					taskBaseName = strings.TrimPrefix( string(watchEvent.Kv.Key), config.Cfg.TaskDir)
					task = &common.Task{TaskName: taskBaseName}
					// 推给scheduler调度器一个删除事件
					taskEvent = &common.TaskEvent{
						CurEvent: common.EventDelete,
						CurTask:  task,
					}
					break
				}

				// 将事件推给scheduler调度器
				scheduler.Sched.PushTaskEvent(taskEvent)
			}
		}
	}()

	return nil
}

func (This *TaskManager) WatchKiller() (err error) {
	// 和WatchTasks类似，但是不需要监听指定版本后的变化

	// 开一个协程持续监听该目录的的变化事件
	go func() {
		var (
			watchRespChan				clientv3.WatchChan
			watchResp					clientv3.WatchResponse
			watchEvent					*clientv3.Event

			taskBaseName				string
			task						*common.Task
			taskEvent					*common.TaskEvent
		)
		watchRespChan = This.Watcher.Watch(context.TODO(), config.Cfg.KillerDir, clientv3.WithPrefix())

		for watchResp = range watchRespChan{
			for _, watchEvent = range watchResp.Events{
				switch watchEvent.Type {
				// 强杀任务
				case clientv3.EventTypePut:
					// 获取任务名(不包括目录名),推给scheduler调度器一个强杀事件
					taskBaseName = strings.TrimPrefix( string(watchEvent.Kv.Key), config.Cfg.KillerDir)
					task = &common.Task{TaskName: taskBaseName}
					// 推给scheduler调度器一个强杀事件
					taskEvent = &common.TaskEvent{
						CurEvent: common.EventKill,
						CurTask:  task,
					}
					// 将事件推给scheduler调度器
					scheduler.Sched.PushTaskEvent(taskEvent)
					break

				// 任务已经被杀死(租约过期)
				case clientv3.EventTypeDelete:
					// master中设置该key的租约已经过期了
					break
				}

			}
		}
	}()


	return nil
}


// 任务管理器单例
var(
	TM 				*TaskManager
)

// 初始化单例
func InitTaskManager() (err error) {
	if TM == nil {
		var (
			etcdConfig clientv3.Config
			tm         TaskManager
		)
		etcdConfig = clientv3.Config{
			Endpoints:   config.Cfg.Endpoints,
			DialTimeout: config.Cfg.DialTimeout,
			DialOptions:  []grpc.DialOption{
				grpc.WithBlock(),
			},
		}
		
		// 建立连接
		if tm.Client , err = clientv3.New(etcdConfig); err != nil{
			return err
		}
		
		// 得到KV、Lease、Watcher的子集
		tm.KV = clientv3.NewKV(tm.Client)
		tm.Lease = clientv3.NewLease(tm.Client)
		tm.Watcher = clientv3.NewWatcher(tm.Client)

		// 赋值单例
		TM = &tm

		// 设置分布式锁对应的etcd的KV和Lease的API子集
		lock.SetEtcdKvAndLeaseAPI(tm.KV, tm.Lease)
	}
	return nil
}
