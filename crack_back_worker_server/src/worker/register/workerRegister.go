package register

import (
	"context"
	"crack_back/src/common"
	"crack_back/src/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"net"
	"path"
	"time"
)

type Register struct {
	client 				*clientv3.Client
	kv 					clientv3.KV
	lease				clientv3.Lease
	workerIP			string
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

	err = common.ERROR_IP_NOT_FOUND
	return
}

// 服务注册(租约+续租)
func (This *Register) keepAlive ()  {
	var(
		err 							error
		workerKey						string
		leaseGrantResp					*clientv3.LeaseGrantResponse
		leaseID							clientv3.LeaseID
		leaseKeepAliveChan				<-chan *clientv3.LeaseKeepAliveResponse
		leaseKeepAliveResp				*clientv3.LeaseKeepAliveResponse

		op								clientv3.Op

		cancelCtx						context.Context
		cancelFunc						context.CancelFunc
	)

	for {
		workerKey = path.Join(config.Cfg.WorkersDir, This.workerIP)
		cancelCtx, cancelFunc = context.WithCancel(context.TODO())

		if leaseGrantResp, err = This.lease.Grant(cancelCtx, 5); err != nil {
			goto TRY_AGAIN
		}

		leaseID = leaseGrantResp.ID

		if leaseKeepAliveChan, err = This.client.KeepAlive(cancelCtx, leaseID); err != nil {
			goto TRY_AGAIN
		}

		op = clientv3.OpPut(workerKey, "", clientv3.WithLease(leaseID))
		if _, err = This.client.Do(cancelCtx, op); err != nil {
			goto TRY_AGAIN
		}

		// 处理续租应答
		for {
			select {
			case leaseKeepAliveResp = <-leaseKeepAliveChan:
				if leaseKeepAliveResp == nil {
					// 续租失败
					goto TRY_AGAIN
				}
			}
		}

	TRY_AGAIN:
		cancelFunc()
		time.Sleep(5 * time.Second)
	}
}

// 注册器单例
var (
	WorkerRegister 			*Register
)

func InitWorkerRegister() (err error) {
	if WorkerRegister == nil{
		var(
			client 				*clientv3.Client
			etcdConfig 			clientv3.Config
			ip 					string
		)

		if ip, err = getIP(); err != nil{
			return
		}

		etcdConfig = clientv3.Config{
			Endpoints:   config.Cfg.Endpoints,
			DialTimeout: config.Cfg.DialTimeout,
			DialOptions:  []grpc.DialOption{
				grpc.WithBlock(),
			},
		}

		// 连接etcd
		if client, err = clientv3.New(etcdConfig); err != nil{
			return err
		}

		// 赋值单例
		WorkerRegister = &Register{
			client:   client,
			kv:       clientv3.NewKV(client),
			lease:    clientv3.NewLease(client),
			workerIP: ip,
		}

		// 注册服务
		go WorkerRegister.keepAlive()
	}
	return nil
}