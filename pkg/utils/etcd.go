package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.etcd.io/etcd/client"
)

// WithFixedIntervalRetry calls the function `fn` `retry` times,
// waiting for the duration indicated by the `wait` parameter between
// two calls.
func WithFixedIntervalRetry(fn func() error, retry uint, wait time.Duration) error {
	var err error
	var x uint
	for x = 0; x < retry; x++ {
		if err = fn(); err != nil {
			time.Sleep(wait)
		} else {
			return nil
		}
	}
	return err
}

// GetEtcdKey configure an etcd client, connects, check the health
// of the quorum and return the object used to interact with it.
func GetEtcdKey(etcdEndpoints []string) (*client.KeysAPI, error) {
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
		l = log.Warn()
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
		l := log.Warn()
		l.Str("error", err.Error())
		for i, e := range etcdEndpoints {
			l.Str(fmt.Sprintf("endpoint-%d", i), e)
		}
		l.Msg("etcd healthcheck failed.")
		return nil, err
	}

	log.Debug().Msg("etcd connection established.")
	return &kapi, nil
}
