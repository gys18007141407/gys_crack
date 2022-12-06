package elector

import (
	"context"
	"crack_front/src/config"
	"crack_front/src/master/alerter"
	"crack_front/src/master/logger"
	"errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"net"
	"time"
)

// 选举器

type Elector struct {
	client 			*clientv3.Client
	kv 				clientv3.KV
	lease			clientv3.Lease
	isLeader		bool

	lockKey			string
	lockValue		string
}

func getIP() (ipv4 string, err error) {
	var(
		adders 				[]net.Addr
		addr 				net.Addr

		ip					*net.IPNet
		ok					bool
	)
	// 获取所有网卡地址
	if adders, err = net.InterfaceAddrs(); err != nil{
		return
	}

	// 找到一个合适的IP地址
	for _,addr = range adders{
		// 类型断言(是否是IP地址)
		if ip, ok = addr.(*net.IPNet); ok && !ip.IP.IsLoopback() {
			if ip.IP.To4() != nil{
				ipv4 = ip.IP.String()
				return
			}
		}
	}
	return "", errors.New("没有找到合适的IPv4的地址")
}

func (This *Elector) IsLeader() bool {
	return This.isLeader
}

// 竞争leader
func (This *Elector) loop ()  {
	var(
		err						error
		leaseID					clientv3.LeaseID
		leaseGrantResp			*clientv3.LeaseGrantResponse
		leaseKeepAliveRespChan	<-chan *clientv3.LeaseKeepAliveResponse
		leaseKeepAliveClose		chan bool

		txn						clientv3.Txn
		txnResp					*clientv3.TxnResponse

		ctx						context.Context
		cancelFunc				context.CancelFunc
	)

	for {
		// 续租关闭通知
		leaseKeepAliveClose = make(chan bool, 1)

		// 申请一个租约
		if leaseGrantResp, err = This.lease.Grant(context.TODO(), 5); err != nil {
			goto FAIL_GET_LOCK
		}
		leaseID = leaseGrantResp.ID

		// 手动取消续租上下文
		ctx, cancelFunc = context.WithCancel(context.TODO())

		// 续租
		if leaseKeepAliveRespChan, err = This.lease.KeepAlive(ctx, leaseID); err != nil{
			goto FAIL_GET_LOCK
		}

		// 响应续租
		go func() {
			var (
				keepAliveResp		*clientv3.LeaseKeepAliveResponse
			)
			KEEPALIVE:
			for{
				select {
				case keepAliveResp = <-leaseKeepAliveRespChan:
					if keepAliveResp == nil{	// 续租被取消
						break KEEPALIVE
					}
					break
				}
			}
			// 通知续租关闭
			leaseKeepAliveClose <- true
		}()

		// 创建事务
		txn = This.kv.Txn(context.TODO())

		// 定义事务
		txn = txn.If(clientv3.Compare(clientv3.CreateRevision(This.lockKey), "=", 0)).Then(
			clientv3.OpPut(This.lockKey, This.lockValue, clientv3.WithLease(leaseID))).Else(
				clientv3.OpGet(This.lockKey))

		// 提交事务
		if txnResp, err = txn.Commit(); err != nil{
			goto FAIL_GET_LOCK
		}

		// 查看事务状态
		if txnResp.Succeeded{
			This.isLeader = true
		}else{
			goto FAIL_GET_LOCK
		}

		// 成为了Leader
		// 警报器随着FAIL_GET_LOCK的cancelFunc而关闭
		logger.Logger.InfoLog("I am Leader")
		go alerter.Alert.Start(ctx)

		// 监听Leader退出
		select {
		case <-leaseKeepAliveClose:   // 续租关闭，放弃Leader
			goto FAIL_GET_LOCK
		}

	// 重新竞争Leader
	FAIL_GET_LOCK:
		This.isLeader = false
		cancelFunc()
		_, _ = This.lease.Revoke(context.TODO(), leaseID)
		// 等一会儿重试
		time.Sleep(time.Second)
	}
}


// 选举器单例
var(
	Elect		*Elector
)

// 初始化单例
func InitElector() (err error) {
	var(
		etcdCfg		clientv3.Config
		client		*clientv3.Client
		ip			string
	)
	etcdCfg = clientv3.Config{
		Endpoints: config.Cfg.Endpoints,
		DialTimeout: config.Cfg.DialTimeout,
		DialOptions:  []grpc.DialOption{
			grpc.WithBlock(),
		},
	}
	if client, err = clientv3.New(etcdCfg); err != nil{
		return
	}

	ip, _ = getIP()

	// 赋值单例
	Elect = &Elector{
		client:    client,
		kv:        clientv3.NewKV(client),
		lease:     clientv3.NewLease(client),
		isLeader:  false,
		lockKey:   config.Cfg.LeaderKey,
		lockValue: ip,
	}

	// 竞争leader
	go Elect.loop()

	return
}