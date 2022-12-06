package alerter

import (
	"context"
	"crack_back/src/config"
	"crack_back/src/worker/logger"
	"github.com/Shopify/sarama"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// 警报器

type Alerter struct {
	handler					groupHandler

	warnTopic				string
	kafkaClient 			sarama.Client
	producer 				sarama.AsyncProducer
	consumerGroup 			sarama.ConsumerGroup
}

// 消费者组句柄[ sarama.ConsumerGroup 接口，实现下面三个方法，作为自定义 ConsumerGroup ]
type groupHandler struct {
	client 				*clientv3.Client
	lease				clientv3.Lease
	kv 					clientv3.KV
}
// 在获得新 session 后， 进行具体的消费逻辑之前执行 Setup
func (This groupHandler)Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}
// 在 session 结束前, 当所有 ConsumeClaim 协程都退出时，执行 Cleanup
func (This groupHandler)Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}
// 具体的消费逻辑
func (This groupHandler)ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim)  error {
	var(
		err 				error
		message 			*sarama.ConsumerMessage
		leaseGrantResp 		*clientv3.LeaseGrantResponse
		op					clientv3.Op
	)
	// 消费kafka中warnTopic下的信息
	for message = range claim.Messages(){
		// 放入etcd中供master的Leader监听，租约为1s
		if leaseGrantResp, err = This.lease.Grant(context.TODO(), 1); err != nil{
			// 获取租约异常，跳过该警报消息
			continue
		}
		op = clientv3.OpPut(string(message.Key), string(message.Value), clientv3.WithLease(leaseGrantResp.ID))
		_, _ = This.kv.Do(context.TODO(), op)
	}
	return nil
}

// 推送警报消息到消息队列[简单地使用channel在宕机时可能会导致消息丢失]
func (This *Alerter) Push(key []byte, value []byte)  {
	var(
		message 				*sarama.ProducerMessage
	)
	message = &sarama.ProducerMessage{
		Topic:     This.warnTopic,
		Key:       sarama.StringEncoder(key),
		Value:     sarama.ByteEncoder(value),
	}
	// 往MQ中推送消息
	This.producer.Input() <- message
}

// 从消息队列中取出来放入etcd中[etcd插入太慢，因此异步插入警报信息，先插入消息队列，然后从消息队列中取消息插入etcd中。consumer]
func (This *Alerter) moveToEtcd()  {
	var(
		err 				error
	)
	if err = This.consumerGroup.Consume(context.TODO(), []string{This.warnTopic}, This.handler); err != nil{
		// 停止转发工作时记录日志
		logger.Logger.WarnLog("Alter has stopped consuming message from kafka! err=", err)
	}
}

// 警报器单例
var (
	Alert			*Alerter
)

// 初始化警报器
func InitAlerter() (err error) {
	if Alert == nil {
		var (
			etcdCfg 		clientv3.Config
			client  		*clientv3.Client

			kafkaConfig		*sarama.Config
			kafkaClient		sarama.Client
			producer		sarama.AsyncProducer
			consumerGroup	sarama.ConsumerGroup
		)

		// 连接etcd客户端
		etcdCfg = clientv3.Config{
			Endpoints:   config.Cfg.Endpoints,
			DialTimeout: config.Cfg.DialTimeout,
			DialOptions:  []grpc.DialOption{
				grpc.WithBlock(),
			},
		}
		if client, err = clientv3.New(etcdCfg); err != nil {
			return
		}

		// 连接kafka客户端
		kafkaConfig = sarama.NewConfig()
		kafkaConfig.Net.DialTimeout = config.Cfg.KafkaTimeout
		kafkaConfig.Producer.Partitioner = sarama.NewRandomPartitioner

		if kafkaClient, err = sarama.NewClient(config.Cfg.BrokerAddrs, kafkaConfig); err != nil{
			return
		}
		if producer, err = sarama.NewAsyncProducerFromClient(kafkaClient); err != nil{
			return
		}
		if consumerGroup, err = sarama.NewConsumerGroupFromClient(config.Cfg.GroupName, kafkaClient); err != nil{
			return err
		}

		// 初始化单例
		Alert = &Alerter{
			handler: groupHandler{
				client:    client,
				lease:     clientv3.NewLease(client),
				kv:        clientv3.NewKV(client),
			},
			warnTopic:     config.Cfg.WarnTopic,
			kafkaClient:   kafkaClient,
			producer:      producer,
			consumerGroup: consumerGroup,
		}

		// 开启协程去消费消息队列中的警报任务
		go Alert.moveToEtcd()
	}
	return
}