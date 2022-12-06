package common

import "errors"

var (
	ERROR_DELETE_EXECSTATUS 					error = errors.New("错误地删除任务，任务执行状态表中已经不包含该任务,该任务可能已经被强制杀死了")
	ERROR_TASKEVENT								error = errors.New("该任务触发了一个错误的事件类型")
	ERROR_SCHEDULEPLAN							error = errors.New("错误地删除调度计划，调度计划表中已经不包含该任务")
	ERROR_KILLTASK								error = errors.New("错误地强杀任务，该任务未正在执行")
	ERROR_TIMEOUT								error = errors.New("该任务由于执行超时被杀死")

	ERROR_LOCK_REQUIRED  						error = errors.New("该锁已经被占用,加锁失败")
	ERROR_TXN_COMMIT  							error = errors.New("提交事务失败")

	ERROR_IP_NOT_FOUND							error = errors.New("未找到一个非环回地址的IP地址")
)
