package workerManager

import (
	"context"
	"crack_front/src/config"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"strings"
)

type WorkerManager struct {
	client 				*clientv3.Client
	kv 					clientv3.KV
}

func (This *WorkerManager) GetWorkers() (workers []string, err error) {
	var (
		op				clientv3.Op
		opResp			clientv3.OpResponse
		kvPair			*mvccpb.KeyValue
		ip 				string
	)
	workers = make([]string, 0, 32)
	op = clientv3.OpGet(config.Cfg.WorkersDir, clientv3.WithPrefix())
	if opResp, err = This.kv.Do(context.TODO(), op); err != nil{
		return
	}

	// 提取KEY中的IP
	for _, kvPair = range opResp.Get().Kvs{
		ip = strings.TrimPrefix(string(kvPair.Key), config.Cfg.WorkersDir)
		workers = append(workers, ip)
	}

	return
}

var (
	WM					*WorkerManager
)

func InitWorkerManager() (err error) {
	if WM == nil{
		var(
			client 			*clientv3.Client
			etcdConfig		clientv3.Config
		)
		etcdConfig = clientv3.Config{
			Endpoints:   config.Cfg.Endpoints,
			DialTimeout: config.Cfg.DialTimeout,
		}
		if client, err = clientv3.New(etcdConfig); err != nil{
			return
		}

		WM = &WorkerManager{
			client: client,
			kv:     clientv3.NewKV(client),
		}
	}
	return nil
}