package config

import (
	"github.com/Unknwon/goconfig"
	"strconv"
	"strings"
	"time"
)

// 加载的配置
type Config struct {
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
	LockDir				string
	WarnDir				string
	FinishDir			string
	FailDir				string

	// worker
	WorkersDir			string

	// database
	DatabaseURI			string
	ConnectTimeOut		time.Duration
	DatabaseName 		string
	Collection 			string
	BatchSize 			int
	CommitInterval		time.Duration

	// MQ
	BrokerAddrs			[]string
	KafkaTimeout		time.Duration
	WarnTopic			string
	GroupName 			string
}

// 配置的单例
var(
	Cfg 	*Config
)

// 初始化配置
func InitConfig(configFile string) (err error) {
	if Cfg == nil {
		var (
			cf     *goconfig.ConfigFile
			config Config
		)
		// 读取配置文件
		if cf, err = goconfig.LoadConfigFile(configFile); err != nil {
			return err
		}

		// 初始化
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

		if err = initMongoDBConfig(cf, &config); err != nil{
			return err
		}

		if err = initKafkaConfig(cf, &config); err != nil{
			return err
		}

		Cfg = &config
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
		baseDir 			string
		killerDir 			string
		lockDir				string
		warnDir				string
		finishDir			string
		failDir				string
	)

	if baseDir, err = cf.GetValue("task", "TaskDir"); err != nil{
		return err
	}
	if killerDir, err = cf.GetValue("task", "KillerDir"); err != nil{
		return err
	}
	if lockDir, err = cf.GetValue("task", "LockDir"); err != nil{
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

	config.TaskDir = baseDir
	config.KillerDir = killerDir
	config.LockDir = lockDir
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

// 初始mongodb配置
func initMongoDBConfig(cf *goconfig.ConfigFile, config *Config) (err error) {
	var(
		dbURI 					string
		connectTimeOutStr 		string
		connectTimeOut			int
		dbName					string
		collection 				string
		batchSizeStr			string
		batchSize 				int
		commitIntervalStr		string
		commitInterval			int
	)

	if dbURI, err = cf.GetValue("MongoDB", "DatabaseURI"); err != nil{
		return err
	}
	if connectTimeOutStr, err = cf.GetValue("MongoDB", "ConnectTimeOut"); err != nil{
		return err
	}
	if dbName, err = cf.GetValue("MongoDB", "DatabaseName"); err != nil{
		return err
	}
	if collection, err = cf.GetValue("MongoDB", "Collection"); err != nil{
		return err
	}
	if batchSizeStr, err = cf.GetValue("MongoDB", "BatchSize"); err != nil{
		return err
	}
	if commitIntervalStr, err = cf.GetValue("MongoDB", "CommitInterval"); err != nil{
		return err
	}

	if connectTimeOut, err = strconv.Atoi(connectTimeOutStr); err != nil{
		return err
	}
	if batchSize, err = strconv.Atoi(batchSizeStr); err != nil{
		return err
	}
	if commitInterval, err = strconv.Atoi(commitIntervalStr); err != nil{
		return err
	}

	config.DatabaseURI = dbURI
	config.ConnectTimeOut = time.Duration(connectTimeOut)*time.Millisecond
	config.DatabaseName = dbName
	config.Collection = collection
	config.BatchSize = batchSize
	if config.BatchSize < 1{
		config.BatchSize = 1
	}
	config.CommitInterval = time.Duration(commitInterval)*time.Millisecond

	return nil
}


// 初始化kafka配置
func initKafkaConfig(cf *goconfig.ConfigFile, config *Config) (err error) {
	var(
		brokerAddrs 				string
		timeoutStr					string
		timeout						int
		warnTopic 					string
		groupName					string
	)

	if brokerAddrs, err = cf.GetValue("kafka", "BrokerAddrs"); err != nil{
		return err
	}

	if timeoutStr, err = cf.GetValue("kafka", "KafkaTimeout"); err != nil{
		return err
	}

	if warnTopic, err = cf.GetValue("kafka", "WarnTopic"); err != nil{
		return err
	}

	if groupName, err = cf.GetValue("kafka", "GroupName"); err != nil{
		return err
	}

	if timeout, err = strconv.Atoi(timeoutStr); err != nil{
		return err
	}


	config.BrokerAddrs = strings.Split(brokerAddrs, ",")
	config.KafkaTimeout = time.Duration(timeout)*time.Millisecond
	config.WarnTopic = warnTopic
	config.GroupName = groupName

	return nil
}
