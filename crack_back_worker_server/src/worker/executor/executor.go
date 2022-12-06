package executor

import (
	"context"
	"crack_back/src/common"
	"crack_back/src/worker/lock"
	"os/exec"
	"path"
	"strconv"
	"time"
)

// 任务执行器
type Executor struct {

}

// 绑定的方法
// 执行任务
func (This *Executor) ExecTask (taskExecStatus *common.TaskExecStatus, callback func(result *common.TaskExecResult)())  {
	// 创建一个协程来执行该任务
	go func() {
		var (
			err						error
			output					[]byte
			taskExecResult			*common.TaskExecResult

			cmd						*exec.Cmd
			task					*common.Task
			userTask				string
			taskLock 				*lock.TaskLock
		)
		task = taskExecStatus.CurTask
		userTask = path.Join(path.Join(task.TaskType, strconv.Itoa(int(task.UserId)), task.TaskName))

		// 分布式锁
		taskLock = lock.NewLock(userTask)
		if err = taskLock.TryLock(); err != nil{
			goto CREATE_EXEC_RESULT
		}

		// 新建cmd调用python程序
		cmd = exec.CommandContext(taskExecStatus.CancelCtx, "python", "-c", taskExecStatus.CurTask.TaskName)

		taskExecStatus.ExecTime = time.Now()
		// 执行cmd
		output, err = cmd.CombinedOutput()
		taskExecStatus.FinishTime = time.Now()

CREATE_EXEC_RESULT:
		// 执行结束,解锁
		taskLock.UnLock()

		// CancelCtx超时而退出
		if taskExecStatus.CancelCtx.Err() == context.DeadlineExceeded{
			err = common.ERROR_TIMEOUT
		}

		// 执行结果信息
		taskExecResult = &common.TaskExecResult{
			CurTaskExecStatus: taskExecStatus,
			CurTaskOutput:     output,
			CurTaskError:      err,
		}

		// 将执行结果传给调度器(协程通信)
		// scheduler.Sched.PushTaskExecResult(taskExecResult)  // import cycle
		callback(taskExecResult)
	}()
}

// 任务执行器单例
var (
	Exec		*Executor
)

func init()  {
	Exec = &Executor{}
}