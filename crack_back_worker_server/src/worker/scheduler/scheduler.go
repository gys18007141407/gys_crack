package scheduler

import (
	"context"
	"crack_back/src/common"
	"crack_back/src/config"
	"crack_back/src/worker/alerter"
	"crack_back/src/worker/executor"
	"crack_back/src/worker/logger"
	"crack_back/src/worker/notifier"
	"crack_back/src/worker/taskLogger"
	"encoding/json"
	"errors"
	"path"
	"strconv"
	"time"
)

// 任务调度器
type Scheduler struct {
	// 任务事件队列
	EventChan 			chan *common.TaskEvent
	// 任务执行状态表 user_id/task_name --> status
	ExecStatus			map[string]*common.TaskExecStatus
	// 任务执行结果队列
	ExecResultChan		chan *common.TaskExecResult
}

// 调度器的事件循环:监听调度器管道
func (This *Scheduler) loop ()  {
	var(
		err 				error
		taskEvent			*common.TaskEvent
		taskExecResult		*common.TaskExecResult
	)

	// 处理到来的调度事件
	for{
		select {
		case taskEvent = <-This.EventChan:
			if err = This.solveTaskEvent(taskEvent); err != nil{
				logger.Logger.InfoLog(err)
			}
			break
		case taskExecResult = <-This.ExecResultChan:
			if err = This.solveTaskExecResult(taskExecResult); err != nil{
				logger.Logger.InfoLog(err)
			}
			break
		}
	}
}

// 调度器处理任务事件
func (This *Scheduler) solveTaskEvent(taskEvent *common.TaskEvent) (err error)  {
	var(
		taskExecStatus	*common.TaskExecStatus
		ok				bool
	)

	switch taskEvent.CurEvent {
	case common.EventSave:
		// 任务到达，准备抢锁执行
		This.ExecTask(taskEvent.CurTask)
		break
	case common.EventDelete:
		// 删除任务事件

		break
	case common.EventKill:
		// 强杀该任务(取消Command执行, CancelFunc())
		if taskExecStatus, ok = This.ExecStatus[taskEvent.CurTask.TaskName]; !ok{
			return common.ERROR_KILLTASK
		}else{
			taskExecStatus.DoCancelFunc()
			// delete(This.ExecStatus, taskEvent.CurTask.TaskName)
		}
		break

	default:
		return common.ERROR_TASKEVENT
	}
	return nil
}

// 调度器处理任务完成信息[写任务日志，报警任务,完成任务，失败任务]
func (This *Scheduler) solveTaskExecResult(taskExecResult *common.TaskExecResult) (err error) {
	var (
		task				*common.Task
		userTask			string
		ok					bool
		warnMessage			*common.WarnMessage
		warnMessageKey		[]byte
		warnMessageValue	[]byte
	)
	task = taskExecResult.CurTaskExecStatus.CurTask
	userTask = path.Join(path.Join(task.TaskType, strconv.Itoa(int(task.UserId)), task.TaskName))
	logger.Logger.InfoLog(userTask, "output=", string(taskExecResult.CurTaskOutput), "err=", taskExecResult.CurTaskError)

	if _, ok = This.ExecStatus[userTask]; !ok{
		return common.ERROR_DELETE_EXECSTATUS
	}
	delete(This.ExecStatus, userTask)

	// 写日志[加锁失败很正常，这类日志可忽略，否则在worker很多的情况下将导致大量加锁失败的日志]
	if taskExecResult.CurTaskError != common.ERROR_LOCK_REQUIRED {
		taskLogger.Logger.PushTaskLog(This.NewTaskLog(taskExecResult))

		// 某类错误将触发报警[这里是除了加锁失败的所有错误都将报警]
		if taskExecResult.CurTaskError != nil{
			// 通知任务失败,往fail目录下插入key
			if err = notifier.Notify.NotifyTaskFailed(task); err != nil {
				logger.Logger.WarnLog(userTask, "notify Finish failed, err=", err.Error())
			}

			// 报警
			warnMessage = This.NewWarnMessage(taskExecResult)
			warnMessageKey = []byte(path.Join(config.Cfg.WarnDir, userTask))
			if warnMessageValue, err = json.Marshal(warnMessage); err != nil{
				return err
			}
			alerter.Alert.Push(warnMessageKey, warnMessageValue)


		} else {
			// 通知任务成功,往finish目录下插入key
			if err = notifier.Notify.NotifyTaskFinished(task); err != nil {
				logger.Logger.WarnLog(userTask, "notify Fail failed, err=", err.Error())
			}
		}
	}

	return nil
}


// 执行任务
func (This *Scheduler) ExecTask (task *common.Task) (err error) {
	var(
		userTask			string
		ok						bool						// isExecuting
		taskExecStatus 			*common.TaskExecStatus
	)
	userTask = path.Join(path.Join(task.TaskType, strconv.Itoa(int(task.UserId))), task.TaskName)

	if taskExecStatus, ok = This.ExecStatus[userTask]; ok{
		logger.Logger.InfoLog("任务正在执行!")
		err = errors.New("任务正在执行!")
		return
	}

	// 新建任务执行状态并保存
	taskExecStatus = This.NewTaskExecStatus(task)
	This.ExecStatus[userTask] = taskExecStatus

	// 将任务提交给执行器
	// executor.Exec.ExecTask(taskExecStatus)			// import cycle
	executor.Exec.ExecTask(taskExecStatus, This.PushTaskExecResult)
	return nil
}


// 给调度器推送任务事件
func (This *Scheduler) PushTaskEvent (taskEvent *common.TaskEvent)  {
	select {
	case This.EventChan <- taskEvent:
		break
	default:
		logger.Logger.InfoLog("任务事件队列满,已经忽略该任务事件")
		break
	}
}

// 给调度器推送任务执行结果
func (This *Scheduler) PushTaskExecResult (taskExecResult *common.TaskExecResult)  {
	This.ExecResultChan <- taskExecResult
}


// 根据TaskSchedule任务调度计划创建新的TaskExecStatus任务执行状态信息
func (This *Scheduler) NewTaskExecStatus (task *common.Task) (taskExecStatus *common.TaskExecStatus) {
	var(
		ctx 			context.Context
		cancelFunc 		context.CancelFunc
	)

	// 超时上下文
	if task.TaskTimeOut > 0 {
		ctx, cancelFunc = context.WithTimeout(context.TODO(), time.Duration(task.TaskTimeOut) * time.Second)
	}else {
		ctx, cancelFunc = context.WithCancel(context.TODO())
	}

	taskExecStatus = &common.TaskExecStatus{
		CurTask:          task,
		ExecTime:         time.Time{},
		FinishTime:       time.Time{},
		CancelCtx:        ctx,
		DoCancelFunc:     cancelFunc,
	}
	return
}


// 创建新的任务日志
func (This *Scheduler) NewTaskLog(taskExecResult *common.TaskExecResult) (taskLog *common.TaskLog) {
	taskLog = &common.TaskLog{
		TaskName:         taskExecResult.CurTaskExecStatus.CurTask.TaskName,
		TaskId: 		  taskExecResult.CurTaskExecStatus.CurTask.TaskId,
		TaskType: 		  taskExecResult.CurTaskExecStatus.CurTask.TaskType,
		UserId:			  taskExecResult.CurTaskExecStatus.CurTask.UserId,
		TaskOutput:       string(taskExecResult.CurTaskOutput),
		ExecTime:         taskExecResult.CurTaskExecStatus.ExecTime.UnixNano() / 1000 / 1000,
		FinishTime:       taskExecResult.CurTaskExecStatus.FinishTime.UnixNano() / 1000 / 1000,
	}
	if taskExecResult.CurTaskError != nil{
		taskLog.TaskError = taskExecResult.CurTaskError.Error()
	}
	return
}

// 创建新的警报消息
func (This *Scheduler) NewWarnMessage(taskExecResult *common.TaskExecResult) (warnMessage *common.WarnMessage) {
	warnMessage = &common.WarnMessage{
		TaskName:         	taskExecResult.CurTaskExecStatus.CurTask.TaskName,
		TaskId: 		  	taskExecResult.CurTaskExecStatus.CurTask.TaskId,
		TaskType: 		  	taskExecResult.CurTaskExecStatus.CurTask.TaskType,
		UserId:			  	taskExecResult.CurTaskExecStatus.CurTask.UserId,
		Message:      		taskExecResult.CurTaskError.Error(),
		GenerateTime: 		taskExecResult.CurTaskExecStatus.FinishTime.UnixNano(),
	}
	return
}

// 任务调度器单例
var (
	Sched		*Scheduler
)

func init()  {
	Sched = &Scheduler{
		EventChan: make(chan *common.TaskEvent, 512),
		ExecStatus: make(map[string]*common.TaskExecStatus, 512),
		ExecResultChan: make(chan *common.TaskExecResult, 512),
	}

	// 启动任务调度器
	go Sched.loop()
}