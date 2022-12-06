package main

import (
	"crack_back/src/config"
	"crack_back/src/worker/alerter"
	"crack_back/src/worker/logger"
	"crack_back/src/worker/notifier"
	"crack_back/src/worker/register"
	"crack_back/src/worker/taskLogger"
	"crack_back/src/worker/taskManager"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
)

// go本身是多线程的，我们在程序中创建的是协程。为了让go效率最大化，设置线程数量等于内核数量
func init()  {
	var(
		cpus 		int
	)
	cpus = runtime.NumCPU()
	runtime.GOMAXPROCS(cpus)
}

var(
	configFile  			string
)


func main()  {
	var (
		err 					error
		normalQuit				chan error
		forceQuit				chan os.Signal
	)

	// 解析命令行参数
	flag.StringVar(&configFile, "config", "/home/gys/go/src/crack_back_worker_server/src/config/config.ini", "指定配置文件路径")
	flag.Parse()

	// 读取并初始化配置
	if err = config.InitConfig(configFile); err != nil{
		fmt.Println("初始化配置文件错误:", err)
		return
	}
	fmt.Println("crack_back初始化配置文件成功")

	// 初始化日志记录器
	if err = logger.InitLogger(); err != nil{
		fmt.Println("初始化日志记录器错误:", err)
		return
	}
	logger.Logger.InfoLog("crack_back初始化配置文件成功")
	logger.Logger.InfoLog("crack_back始化日志记录器成功")

	// 初始化任务管理器
	if err = taskManager.InitTaskManager(); err != nil{
		fmt.Println("crack_back初始化任务管理器错误:", err)
		logger.Logger.WarnLog(err)
		return
	}
	logger.Logger.InfoLog("crack_back初始化任务管理器成功")

	// 初始化任务日志记录器
	if err = taskLogger.InitLogger(); err != nil{
		fmt.Println("crack_back初始化日志记录器错误:", err)
		logger.Logger.WarnLog(err)
		return
	}
	logger.Logger.InfoLog("crack_back初始化任务执行日志管理器成功")

	// 初始化服务注册器
	if err = register.InitWorkerRegister(); err != nil{
		fmt.Println("crack_back初始化服务注册器错误:", err)
		logger.Logger.WarnLog(err)
		return
	}
	logger.Logger.InfoLog("crack_back初始化服务注册器成功")

	// 初始化警报器
	if err = alerter.InitAlerter(); err != nil{
		fmt.Println("crack_back初始化警报器错误:", err)
		logger.Logger.WarnLog(err)
		return
	}
	logger.Logger.InfoLog("crack_back初始化警报器成功")

	// 初始化通知器
	if err = notifier.InitNotifier(); err != nil{
		fmt.Println("crack_back初始化通知器错误:", err)
		logger.Logger.WarnLog(err)
		return
	}
	logger.Logger.InfoLog("crack_back初始化通知器成功")

	// 开始监听任务目录
	go func() {
		if err = taskManager.TM.WatchTasks(); err != nil{
			normalQuit <- err
			return
		}
	}()
	logger.Logger.InfoLog("crack_back开始监听任务目录")

	// 开始监听强杀目录
	go func() {
		if err = taskManager.TM.WatchKiller(); err != nil{
			normalQuit <- err
			return
		}
	}()
	logger.Logger.InfoLog("crack_back开始监听强杀目录")

	normalQuit = make(chan error, 2)
	forceQuit = make(chan os.Signal, 1)
	signal.Notify(forceQuit, os.Interrupt)

	select {
	case <-forceQuit:
		logger.Logger.WarnLog("强制退出")
		break
	case err = <-normalQuit:
		logger.Logger.WarnLog("程序退出:", err)
		break
	}

}
