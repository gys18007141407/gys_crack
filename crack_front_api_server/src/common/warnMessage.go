package common

// 预警信息
type WarnMessage struct {
	TaskType 					string 		`bson:"task_type" json:"task_type"`				// 任务类型(image, video)
	UserId 						uint 		`bson:"user_id" json:"user_id"`					// 发布该任务的用户id
	TaskName 					string		`bson:"task_name" json:"task_name"`         	// 任务名称

	Message 					string		`bson:"message" json:"message"`					// 警告信息
	GenerateTime 				int64		`bson:"generate_time" json:"generate_time"`		// 信息产生时间
}
