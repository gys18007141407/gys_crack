package taskLogger

import (
	"context"
	"crack_back/src/common"
	"crack_back/src/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type TaskLogger struct {
	mongoClient 				*mongo.Client
	mongoCollection 			*mongo.Collection
	taskLogChan 				chan *common.TaskLog
	batchTimeOutChan			chan *common.TaskLogBatch

	cancelCtx					context.Context
	cancelFunc					context.CancelFunc
}

// 监听日志管道循环
func (This *TaskLogger) loop () {
	var (
		taskLogBatch 			*common.TaskLogBatch
		taskLog 				*common.TaskLog
		batchTimeOut			*common.TaskLogBatch
	)

	for {
		select {
		case taskLog = <-This.taskLogChan:
			if taskLogBatch == nil{
				taskLogBatch = &common.TaskLogBatch{
					Logs:          make([]interface{}, 0, config.Cfg.BatchSize),
					// time.AfterFunc会启动另一个协程执行Func
					// 可能造成loop协程和定时器Func协程同时操作同一个logBatch，导致两次落盘
					// 应该让同一个协程处理日志落盘。故而让Func通知loop协程即可，loop协程掌管所有的logSink。
					AutoSinkTimer: time.AfterFunc(config.Cfg.CommitInterval, func(){
						This.batchTimeOutChan <- taskLogBatch
					}),
				}
			}

			taskLogBatch.Logs = append(taskLogBatch.Logs, taskLog)
			if len(taskLogBatch.Logs) >= config.Cfg.BatchSize{
				// 取消定时器
				taskLogBatch.AutoSinkTimer.Stop()

				// 开启另一个协程日志落盘
				go This.logSink(taskLogBatch)
				// 重置日志批次
				taskLogBatch = nil
			}
			break

		case batchTimeOut = <-This.batchTimeOutChan:
			// 如果batchTimeOut != taskLogBatch说明 taskLogBatch 已经是另外一个batch了
			// taskLogBatch变为另一个 batch 必须经过 taskLogBatch = nil 重置日志批次
			// 则说明该batch已经被落盘了
			if batchTimeOut == taskLogBatch {
				go This.logSink(batchTimeOut)
				taskLogBatch = nil
			}
			break
		}
	}
}

// 日志落盘
func (This *TaskLogger) logSink (taskLogBatch *common.TaskLogBatch)  {
	if taskLogBatch != nil {
		_, _ = This.mongoCollection.InsertMany(context.TODO(), taskLogBatch.Logs)
	}
}

// 写入一个日志
func (This *TaskLogger) PushTaskLog (taskLog *common.TaskLog)  {
	This.taskLogChan <- taskLog
}

// 日志记录器单例
var (
	Logger				*TaskLogger
)

func InitLogger() (err error) {
	if Logger == nil {
		var (
			client *mongo.Client
			ctx        context.Context
			cancelFunc context.CancelFunc
		)

		ctx, cancelFunc = context.WithTimeout(context.TODO(), config.Cfg.ConnectTimeOut)

		if client, err = mongo.Connect(ctx, options.Client().ApplyURI(config.Cfg.DatabaseURI)); err != nil {
			cancelFunc()
			return err
		}

		// 赋值单例
		Logger = &TaskLogger{
			mongoClient:      client,
			mongoCollection:  client.Database(config.Cfg.DatabaseName).Collection(config.Cfg.Collection),
			taskLogChan:      make(chan *common.TaskLog, 1024),
			batchTimeOutChan: make(chan *common.TaskLogBatch, 1024),
			cancelCtx:        ctx,
			cancelFunc:       cancelFunc,
		}

		// 写日志
		go Logger.loop()
	}
	return nil
}

