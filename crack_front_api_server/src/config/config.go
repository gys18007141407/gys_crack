package config

import (
	"github.com/Unknwon/goconfig"
	"strconv"
	"strings"
	"time"
)

// 加载的配置
type Config struct {
	// web
	ListenIP 			string
	ListenPort 			int

	// logger
	LogFilePath 		string
	LogFileName			string
	LogLevel 			string

	// etcd
	Endpoints 			[]string
	DialTimeout 		time.Duration

	// task
	TaskDir 			string
	KillerDir			string
	WarnDir				string
	FinishDir			string
	FailDir				string

	// worker
	WorkersDir			string

	// master
	MastersDir			string
	LeaderKey 			string

	// MongoDB
	MongoDB_DatabaseURI			string
	MongoDB_ConnectTimeOut		time.Duration
	MongoDB_DatabaseName		string

	// MySQL
	MySQL_DataSourceName 		string
}

// 配置的单例
var(
	Cfg 	*Config
)

// 初始化配置
func InitConfig(configFile string) error {
	if Cfg == nil {
		var (
			err    error
			cf     *goconfig.ConfigFile
			config Config
		)
		// 读取配置文件
		if cf, err = goconfig.LoadConfigFile(configFile); err != nil {
			return err
		}

		// 初始化

		if err = initWebConfig(cf, &config); err != nil{
			return err
		}

		if err = initLoggerConfig(cf, &config); err != nil{
			return err
		}

		if err = initEtcdConfig(cf, &config); err != nil{
			return err
		}

		if err = initTaskConfig(cf, &config); err != nil{
			return err
		}

		if err = initWorkerConfig(cf, &config); err != nil{
			return err
		}

		if err = initMasterConfig(cf, &config); err != nil{
			return err
		}

		if err = initMongoDBConfig(cf, &config); err != nil{
			return err
		}

		if err = initMySQLConfig(cf, &config); err != nil{
			return err
		}

		Cfg = &config
	}
	return nil
}

// 初始化web配置
func initWebConfig(cf *goconfig.ConfigFile, config *Config) (err error) {
	var(
		port 		string
	)
	if config.ListenIP, err = cf.GetValue("web", "Ip"); err != nil{
		return err
	}

	if port, err = cf.GetValue("web", "Port"); err != nil{
		return err
	}

	if config.ListenPort, err = strconv.Atoi(port); err != nil{
		return err
	}

	return nil
}

// 初始化logger配置
func initLoggerConfig(cf *goconfig.ConfigFile, config *Config) (err error) {

	if config.LogFilePath, err = cf.GetValue("logger", "LogFilePath"); err != nil{
		return err
	}

	if config.LogFileName, err = cf.GetValue("logger", "LogFileName"); err != nil{
		return err
	}

	if config.LogLevel, err = cf.GetValue("logger", "LogLevel"); err != nil{
		return err
	}
	config.LogLevel = strings.ToLower(config.LogLevel)
	return nil
}


// 初始etcd配置
func initEtcdConfig(cf *goconfig.ConfigFile, config *Config) (err error) {
	var(
		endpoints 			string
		dialTimeoutStr 		string
		dialTimeout 		int
	)

	if endpoints, err = cf.GetValue("etcd", "Endpoints"); err != nil{
		return err
	}

	if dialTimeoutStr, err = cf.GetValue("etcd", "DialTimeout"); err != nil{
		return err
	}

	if dialTimeout, err = strconv.Atoi(dialTimeoutStr); err != nil{
		return err
	}

	config.Endpoints = strings.Split(endpoints, ",")
	config.DialTimeout = time.Duration(dialTimeout)*time.Millisecond

	return nil
}

// 初始task配置
func initTaskConfig(cf *goconfig.ConfigFile, config *Config) (err error) {
	var(
		taskDir 			string
		killerDir 			string
		warnDir				string
		finishDir			string
		failDir				string
	)

	if taskDir, err = cf.GetValue("task", "TaskDir"); err != nil{
		return err
	}
	if killerDir, err = cf.GetValue("task", "KillerDir"); err != nil{
		return err
	}
	if warnDir, err = cf.GetValue("task", "WarnDir"); err != nil{
		return err
	}
	if finishDir, err = cf.GetValue("task", "FinishDir"); err != nil{
		return err
	}
	if failDir, err = cf.GetValue("task", "FailDir"); err != nil{
		return err
	}

	config.TaskDir = taskDir
	config.KillerDir = killerDir
	config.WarnDir = warnDir
	config.FinishDir = finishDir
	config.FailDir = failDir

	return nil
}

// 初始worker配置
func initWorkerConfig(cf *goconfig.ConfigFile, config *Config) (err error) {
	var(
		workersDir			string
	)

	if workersDir, err = cf.GetValue("worker", "WorkersDir"); err != nil{
		return err
	}

	config.WorkersDir = workersDir

	return nil
}

// 初始master配置
func initMasterConfig(cf *goconfig.ConfigFile, config *Config) (err error) {
	var(
		mastersDir			string
		leaderKey			string
	)

	if mastersDir, err = cf.GetValue("master", "MastersDir"); err != nil{
		return err
	}

	if leaderKey, err = cf.GetValue("master", "LeaderKey"); err != nil{
		return err
	}

	config.MastersDir = mastersDir
	config.LeaderKey = leaderKey

	return nil
}


// 初始MongoDB配置
func initMongoDBConfig(cf *goconfig.ConfigFile, config *Config) (err error) {
	var(
		ip 						string
		port 					string
		connectTimeOutStr 		string
		connectTimeOut			int
		dbName					string
	)

	if ip, err = cf.GetValue("MongoDB", "Ip"); err != nil{
		return err
	}
	if port, err = cf.GetValue("MongoDB", "Port"); err != nil{
		return err
	}
	if connectTimeOutStr, err = cf.GetValue("MongoDB", "ConnectTimeOut"); err != nil{
		return err
	}
	if dbName, err = cf.GetValue("MongoDB", "DatabaseName"); err != nil{
		return err
	}

	if _, err = strconv.Atoi(port); err != nil {
		return err
	}
	if connectTimeOut, err = strconv.Atoi(connectTimeOutStr); err != nil{
		return err
	}

	config.MongoDB_DatabaseURI = "mongodb://" + ip + ":" + port
	config.MongoDB_ConnectTimeOut = time.Duration(connectTimeOut)*time.Millisecond
	config.MongoDB_DatabaseName = dbName

	return nil
}

// 初始MySQL配置
func initMySQLConfig(cf *goconfig.ConfigFile, config *Config) (err error) {
	var(
		ip 						string
		port 					string
		user 					string
		password 				string
		connectTimeOut			string
		dbName					string
	)

	if ip, err = cf.GetValue("MySQL", "Ip"); err != nil{
		return err
	}
	if port, err = cf.GetValue("MySQL", "Port"); err != nil{
		return err
	}
	if user, err = cf.GetValue("MySQL", "User"); err != nil{
		return err
	}
	if password, err = cf.GetValue("MySQL", "Password"); err != nil{
		return err
	}
	if connectTimeOut, err = cf.GetValue("MySQL", "ConnectTimeOut"); err != nil{
		return err
	}
	if dbName, err = cf.GetValue("MySQL", "DatabaseName"); err != nil{
		return err
	}

	if _, err = strconv.Atoi(port); err != nil {
		return err
	}
	if _, err = strconv.Atoi(connectTimeOut); err != nil {
		return err
	}

	config.MySQL_DataSourceName = user + ":" + password + "@(" + ip + ":" + port + ")/" + dbName + "?charset=utf8mb4&parseTime=True&loc=Local&timeout=" + connectTimeOut + "ms"

	return nil
}