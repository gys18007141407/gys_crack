package common

type	EventType	int

const (
	_         EventType = iota
	EventSave           // 保存任务事件
	EventDelete         // 删除任务事件
	EventKill           // 强杀任务事件
)

type TaskEvent struct {
	CurEvent EventType
	CurTask  *Task
}
