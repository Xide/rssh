package api

import (
	"github.com/Xide/rssh/pkg/utils"

	"go.etcd.io/etcd/client"
)

type APIExecutor struct {
	etcd client.KeysAPI
}

func NewExecutor(etcdEndpoints []string) (*APIExecutor, error) {
	kapi, err := utils.GetEtcdKey(etcdEndpoints)
	if err != nil {
		return nil, err
	}
	return &APIExecutor{
		*kapi,
	}, nil
}
