package common

import "regexp"

type Task struct {
	TaskType 				string 		`json:"task_type"`			// 任务类型(image, video)
	UserId 					uint 		`json:"user_id"`			// 发布该任务的用户id
	TaskName 				string		`json:"task_name"`         	// 任务名称
	TaskTimeOut 			uint 		`json:"task_time_out"`		// 任务超时时间(s)
}

var (
	ImageType				=			"image"
	VideoType				=			"video"
)

func VerifyTaskName(TaskName string) (ok bool){
	ok, _ = regexp.MatchString("^[a-zA-Z0-9_]{1,16}$", TaskName);
	return ok
}

func VerifyTaskType(TaskType string) (ok bool){
	return TaskType == ImageType || TaskType == VideoType
}