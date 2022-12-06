package common

import "time"

type TaskLog struct {
	TaskType 					string 		`bson:"task_type" json:"task_type"`							// 任务类型(image, video)
	UserId 						int64 		`bson:"user_id" json:"user_id"`								// 发布该任务的用户id
	TaskName 					string		`bson:"task_name" json:"task_name"`         				// 任务名称
	TaskId 						int64 		`bson:"task_id" json:"task_id"`								// 任务id

	TaskOutput					string		`bson:"task_output" json:"task_output"`						// 任务标准输出
	TaskError					string		`bson:"task_error" json:"task_error"`						// 任务错误输出
	ScheduleTime				int64		`bson:"schedule_time" json:"schedule_time"`					// 理论被调度的时间
	RealScheduleTime			int64		`bson:"real_schedule_time" json:"real_schedule_time"`		// 真正被调度的时间
	ExecTime					int64		`bson:"exec_time" json:"exec_time"`							// 执行时间
	FinishTime 					int64		`bson:"finish_time" json:"finish_time"`						// 完成时间
}


type TaskLogBatch struct {
	Logs						[]interface{}
	AutoSinkTimer 				*time.Timer
}