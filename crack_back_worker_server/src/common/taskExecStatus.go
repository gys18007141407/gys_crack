package common

import (
	"context"
	"time"
)

type TaskExecStatus struct {
	CurTask						*Task
	ScheduleTime				time.Time				// 理论被调度的时间
	RealScheduleTime			time.Time				// 真正被调度的时间
	ExecTime					time.Time				// 执行时间
	FinishTime 					time.Time				// 完成时间
	CancelCtx 					context.Context			// 取消上下文
	DoCancelFunc				context.CancelFunc		// 取消任务执行
}
