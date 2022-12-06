package common

// 任务执行后的结果信息

type TaskExecResult struct {
	CurTaskExecStatus 					*TaskExecStatus 	// 任务执行状态信息
	CurTaskOutput						[]byte          	// 任务标准输出
	CurTaskError						error            	// 任务错误输出
}
