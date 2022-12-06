package lock

import (
	"context"
	"crack_back/src/common"
	"crack_back/src/config"
	"crack_back/src/worker/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"path"
)

// 锁对应的etcd的API子集
var (
	kv						clientv3.KV			// 锁对应的etcd的kv API子集
	lease 					clientv3.Lease		// 锁对应的etcd的lease API子集
	assigned				bool
)

// 只能设置一次
func SetEtcdKvAndLeaseAPI(KV clientv3.KV, Lease clientv3.Lease){
	if !assigned {
		kv = KV
		lease = Lease
	}
}

// 分布式锁
type TaskLock struct {
	TaskName 				string  			// 一个任务一把锁

	doCancelFunc			context.CancelFunc	// 该锁对应的etcd的key的续租取消函数
	leaseID			 		clientv3.LeaseID	// 该锁对应的etcd的key的租约ID
	locked 					bool
}



// 创建一把锁
func NewLock(taskName string) (taskLock *TaskLock){
	taskLock = &TaskLock{
		TaskName: taskName,
	}
	return
}

// 判断是否已经加锁
func (This*TaskLock) Locked () bool {
	return This.locked
}

// 加锁
func (This *TaskLock) TryLock () (err error) {
	var (
		leaseGrantResp 			*clientv3.LeaseGrantResponse
		leaseID					clientv3.LeaseID
		leaseKeepAliveRespChan	<-chan *clientv3.LeaseKeepAliveResponse

		cancelCtx				context.Context
		cancelFunc				context.CancelFunc
		txn						clientv3.Txn
		txnResp					*clientv3.TxnResponse

		lockKey					string
	)

	// 申请租约
	if leaseGrantResp, err = lease.Grant(context.TODO(), 5); err != nil{
		return
	}
	// 获得租约的ID
	leaseID	= leaseGrantResp.ID

	// 用于手动取消续租的上下文
	cancelCtx, cancelFunc = context.WithCancel(context.TODO())

	// 续租
	if leaseKeepAliveRespChan, err = lease.KeepAlive(cancelCtx, leaseID); err != nil{
		goto FAIL_GET_LOCK
	}

	// 另开一个协程处理续租应答
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
	}()

	// 该锁在etcd中对应的key
	lockKey = path.Join(config.Cfg.LockDir, This.TaskName)

	// 创建事务
	txn = kv.Txn(context.TODO())

	// 定义事务
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).Then(
		clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseID))).Else(
			clientv3.OpGet(lockKey))

	// 提交事务
	if txnResp, err = txn.Commit(); err != nil{
		err = common.ERROR_TXN_COMMIT
		goto FAIL_GET_LOCK
	}

	// 判断事务是否成功
	if txnResp.Succeeded {
		This.doCancelFunc = cancelFunc
		This.leaseID = leaseID
		This.locked = true
	}else{
		err = common.ERROR_LOCK_REQUIRED
		goto FAIL_GET_LOCK
	}

	return nil

FAIL_GET_LOCK:  // 加锁失败
	// 取消续租
	cancelFunc()
	// 取消租约
	// 如果主动取消租约失败, 则等待租约超时释放，无伤大雅
	_, _ = lease.Revoke(context.TODO(), leaseID)
	return
}



// 解锁
func (This *TaskLock) UnLock ()  {
	var(
		err				error
	)
	if This.locked{
		// 取消续租
		This.doCancelFunc()
		// 取消租约
		if _, err = lease.Revoke(context.TODO(), This.leaseID); err != nil{
			logger.Logger.InfoLog("主动取消租约失败, 等待租约超时释放")
		}
	}
}