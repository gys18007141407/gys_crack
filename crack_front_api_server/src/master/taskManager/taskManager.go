package taskManager

import (
	"context"
	"crack_front/src/common"
	"crack_front/src/config"
	"crack_front/src/master/logger"
	"encoding/json"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"path"
	"strconv"
)

// 一个etcd客户端，用来管理任务
type TaskManager struct {
	client 		*clientv3.Client
	kv 			clientv3.KV
	lease 		clientv3.Lease
}

func (This *TaskManager) SaveTask(task *common.Task) (err error) {
	// 将该任务保存到/crack/task/目录下
	var(
		taskKey			string
		taskValue		[]byte
		Op 				clientv3.Op
	)
	taskKey = path.Join(path.Join(path.Join(config.Cfg.TaskDir, task.TaskType), strconv.Itoa(int(task.UserId))), task.TaskName)

	if taskValue, err = json.Marshal(task); err != nil{
		return
	}

	Op = clientv3.OpPut(taskKey, string(taskValue), clientv3.WithPrevKV())
	_, err = This.kv.Do(context.TODO(), Op)
	return
}

func (This *TaskManager) RemoveTask(task *common.Task) (oldTask *common.Task, err error) {
	// 从/crack/task/中删除该任务
	var (
		taskKey string
		Op      clientv3.Op
		OpResp  clientv3.OpResponse
		temp    common.Task
	)

	// 该任务在etcd中的key
	taskKey = path.Join(path.Join(path.Join(config.Cfg.TaskDir, task.TaskType), strconv.Itoa(int(task.UserId))), task.TaskName)

	Op = clientv3.OpDelete(taskKey, clientv3.WithPrevKV())
	if OpResp, err = This.kv.Do(context.TODO(), Op); err != nil{
		return
	}

	// 如果删除成功, 反序列化原来的任务
	if len(OpResp.Del().PrevKvs) != 0{
		if err = json.Unmarshal(OpResp.Del().PrevKvs[0].Value, &temp); err != nil{
			logger.Logger.InfoLog("RemoveTask反序列化错误...已丢弃该错误:", err.Error())
			return nil, nil
		}
		oldTask = &temp
	}
	return
}

// watch某个目录的eventType事件
func (This *TaskManager) WatchDir (dir string, eventType mvccpb.Event_EventType, notifyChan chan<- struct{}, errChan chan<- error)  {
	var (
		err							error
		op							clientv3.Op
		opResp						clientv3.OpResponse
		watchVersion				int64
		watchChan					clientv3.WatchChan
		watchResp					clientv3.WatchResponse
		event						*clientv3.Event
	)

	op = clientv3.OpGet(dir, clientv3.WithPrefix())
	if opResp, err = This.kv.Do(context.TODO(), op); err != nil {
		errChan <- err
		return
	}

	watchVersion = opResp.Get().Header.Revision+1
	watchChan = clientv3.NewWatcher(This.client).Watch(context.TODO(), dir, clientv3.WithPrefix(), clientv3.WithRev(watchVersion))

	// 遍历watch的事件, for退出说明watchChan关闭
	for watchResp = range watchChan{
		for _, event = range watchResp.Events{
			switch event.Type {
			case eventType:					// 想要的事件发生
				// 通知loop协程
				notifyChan <- struct{}{}
			// 不在乎其他事件
			}
		}
	}
	// 期待事件没有发生，watchChan就已经被关闭了 TODO


}
// 返回两个chan通知任务成功或者失败
func (This *TaskManager) WatchTask (task *common.Task) (finishChan <-chan struct{}, failChan <-chan struct{}, errChan <-chan error) {
	var (
		finishDir					string
		failDir						string
		finishChanInternal			= make(chan struct{}, 1)
		failChanInternal			= make(chan struct{}, 1)
		errChanInternal				= make(chan error, 2)
	)

	// 路径
	finishDir = path.Join(path.Join(config.Cfg.FinishDir, task.TaskType), strconv.Itoa(int(task.UserId)))
	failDir = path.Join(path.Join(config.Cfg.FailDir, task.TaskType), strconv.Itoa(int(task.UserId)))

	// 两个协程去watch这两个目录
	go This.WatchDir(finishDir, clientv3.EventTypePut, finishChanInternal, errChanInternal)
	go This.WatchDir(failDir, clientv3.EventTypePut, failChanInternal, errChanInternal)

	return finishChanInternal, failChanInternal, errChanInternal
}

func (This *TaskManager) GetTaskList () (taskList []*common.Task, err error) {
	// 查询/cron/tasks/目录下的所有key
	var (
		Op 				clientv3.Op
		OpResp			clientv3.OpResponse
		kvPair			*mvccpb.KeyValue
		temp			*common.Task
	)
	taskList = make([]*common.Task, 0)

	// 获取任务根目录下的所有任务
	Op = clientv3.OpGet(config.Cfg.TaskDir, clientv3.WithPrevKV(), clientv3.WithPrefix())
	if OpResp, err = This.kv.Do(context.TODO(), Op); err != nil{
		return
	}

	// 遍历Response, 并进行反序列化
	for _, kvPair = range OpResp.Get().Kvs{
		temp = &common.Task{}
		if err = json.Unmarshal(kvPair.Value, temp); err != nil {
			logger.Logger.InfoLog("task反序列化时错误...已丢弃该错误:", err.Error())
			temp.TaskName = "反序列化时发生错误"
		}
		taskList = append(taskList, temp)
	}

	return taskList, nil
}

func (This *TaskManager) KillTask (task *common.Task) (err error){
	// 更新etcd中该任务，将key更新到killer目录下（worker监听该目录，杀死任务）
	var (
		taskKillerKey	string

		leaseGrantResp	*clientv3.LeaseGrantResponse
		leaseID			clientv3.LeaseID

		Op 				clientv3.Op
	)
	taskKillerKey = path.Join(path.Join(path.Join(config.Cfg.KillerDir, task.TaskType), strconv.Itoa(int(task.UserId))), task.TaskName)


	// 申请租约(该key在强杀目录中存在2s，然后被删除。先触发PUT事件，然后触发删除事件)
	if leaseGrantResp, err = TM.lease.Grant(context.TODO(), 2); err != nil{
		return
	}

	// 该租约的ID
	leaseID = leaseGrantResp.ID

	// 置killer标记：将其put到killer目录下，worker监听到后强杀该任务
	Op = clientv3.OpPut(taskKillerKey, "", clientv3.WithLease(leaseID))
	if _, err = This.kv.Do(context.TODO(), Op); err != nil{
		return
	}
	logger.Logger.InfoLog("杀死任务：", task.TaskName)
	return
}


// 任务管理器单例
var(
	TM 				*TaskManager
)

// 初始化单例
func InitTaskManager() (err error) {
	if TM == nil {
		var (
			etcdConfig 		clientv3.Config
			client 			*clientv3.Client
		)
		etcdConfig = clientv3.Config{
			Endpoints:            config.Cfg.Endpoints,
			DialTimeout:          config.Cfg.DialTimeout,
			// 没有grpc.WithBlock()，clientv3.New()将会立即返回，然后在后台连接etcd
			// 这样会造成连不上etcd时也不会返回error
			DialOptions:          []grpc.DialOption{
				grpc.WithBlock(),
			},
		}
		
		// 建立连接
		if client, err = clientv3.New(etcdConfig); err != nil{
			return err
		}

		// 赋值单例
		TM = &TaskManager{
			client: client,
			kv:     clientv3.NewKV(client),
			lease:  clientv3.NewLease(client),
		}
	}
	return nil
}
