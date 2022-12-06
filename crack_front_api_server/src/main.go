package main

import (
	"crack_front/src/config"
	"crack_front/src/master/alerter"
	"crack_front/src/master/elector"
	"crack_front/src/master/logManager"
	"crack_front/src/master/logger"
	"crack_front/src/master/router"
	"crack_front/src/master/taskManager"
	"crack_front/src/master/user"
	"crack_front/src/master/workerManager"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
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
	flag.StringVar(&configFile, "config", "/home/gys/go/src/crack_front_api_server/src/config/config.ini", "指定配置文件路径")
	flag.Parse()

	// 读取并初始化配置
	if err = config.InitConfig(configFile); err != nil{
		fmt.Println("初始化配置文件错误:", err)
		return
	}
	fmt.Println("crack_front初始化配置文件成功")

	// 初始化日志记录器
	if err = logger.InitLogger(); err != nil{
		fmt.Println("初始化日志记录器错误:", err)
		return
	}
	logger.Logger.InfoLog("crack_front初始化配置文件成功")
	logger.Logger.InfoLog("crack_front始化日志记录器成功")

	// 连接MySQL
	if err = user.InitMySQL(); err != nil{
		fmt.Println("crack_front连接MySQL错误:", err)
		logger.Logger.WarnLog(err)
		return
	}
	defer user.CloseMySQL()
	logger.Logger.InfoLog("crack_front连接MySQL成功")

	// 初始化任务管理器
	if err = taskManager.InitTaskManager(); err != nil{
		fmt.Println("crack_front初始化任务管理器错误:", err)
		logger.Logger.WarnLog(err)
		return
	}
	logger.Logger.InfoLog("crack_front初始化任务管理器成功")

	// 初始化任务执行日志管理器
	if err = logManager.InitLogManager(); err != nil{
		fmt.Println("crack_front初始化任务管执行日志管理器错误:", err)
		logger.Logger.WarnLog(err)
		return
	}
	logger.Logger.InfoLog("crack_front初始化任务执行日志管理器成功")

	// 初始化worker集群发现管理器(服务发现)
	if err = workerManager.InitWorkerManager(); err != nil{
		fmt.Println("crack_front初始化worker集群管理器错误:", err)
		logger.Logger.WarnLog(err)
		return
	}
	logger.Logger.InfoLog("crack_front初始化worker集群发现管理器成功")

	// 初始化警报器
	if err = alerter.InitAlerter(); err != nil{
		fmt.Println("crack_front初始化任务警报器错误:", err)
		logger.Logger.WarnLog(err)
		return
	}
	logger.Logger.InfoLog("crack_front初始化警报器成功")

	// 初始化选举器
	if err = elector.InitElector(); err != nil{
		fmt.Println("crack_front初始化选举器错误:", err)
		logger.Logger.WarnLog(err)
		return
	}
	logger.Logger.InfoLog("crack_front初始化选举器成功")

	normalQuit = make(chan error, 1)
	forceQuit = make(chan os.Signal, 1)
	signal.Notify(forceQuit, os.Interrupt)

	// 启动路由
	go func() {
		if err = router.Router.Run(config.Cfg.ListenIP + ":" + strconv.Itoa(config.Cfg.ListenPort)); err != nil{
			normalQuit <- err
			return
		}
	}()
	logger.Logger.InfoLog("crack_front启动成功")

	// 退出
	select {
	case <-forceQuit:
		logger.Logger.WarnLog("强制退出")
		break
	case err = <-normalQuit:
		logger.Logger.WarnLog("程序退出:", err)
	}
}
