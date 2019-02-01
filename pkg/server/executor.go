package api

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.etcd.io/etcd/client"
)

type APIExecutor struct {
	etcd client.KeysAPI
}

func NewExecutor(etcdEndpoints []string) (*APIExecutor, error) {
	cfg := client.Config{
		Endpoints:               etcdEndpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := client.New(cfg)
	l := log.Debug()
	for i, e := range etcdEndpoints {
		l.Str(fmt.Sprintf("endpoint-%d", i), e)
	}
	l.Msg("Connecting to etcd cluster.")
	if err != nil {
		l = log.Fatal()
		l.Str("error", err.Error())
		for i, e := range etcdEndpoints {
			l.Str(fmt.Sprintf("endpoint-%d", i), e)
		}
		l.Msg("etcd connection failed.")
		return nil, err
	}
	kapi := client.NewKeysAPI(c)

	_, err = c.GetVersion(context.Background())
	if err != nil {
		l := log.Fatal()
		l.Str("error", err.Error())
		for i, e := range etcdEndpoints {
			l.Str(fmt.Sprintf("endpoint-%d", i), e)
		}
		l.Msg("etcd healthcheck failed.")
	}

	log.Info().Msg("etcd connection established.")
	return &APIExecutor{
		kapi,
	}, nil
}
