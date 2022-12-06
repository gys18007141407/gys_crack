package common

type Task struct {
	TaskType 				string 		`json:"task_type"`			// 任务类型(image, video)
	UserId 					int64 		`json:"user_id"`			// 发布该任务的用户id
	TaskName 				string		`json:"task_name"`         	// 任务名称
	TaskId 					int64 		`json:"task_id"`			// 任务id

	TaskTimeOut 			int 		`json:"task_time_out"`		// 任务超时时间
}
