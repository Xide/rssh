package api

import (
	"fmt"
	// "regexp"
	"time"
	"context"
	"go.etcd.io/etcd/client"

	"github.com/rs/zerolog/log"
	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)



type Endpoint struct {
	Host string
	Port uint16
}

type APIExecutor struct {
	etcd client.KeysAPI
}

type APIDispatcher struct {
	bindAddr string
	bindPort uint16

	// Agent requests
	RegisterCH chan RegisterRpc

	// Client requests
	ConnectCH chan ConnectRequest
}

func NewExecutor(etcdEndpoints []string) (*APIExecutor, error) {
	cfg := client.Config{
		Endpoints:               etcdEndpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := client.New(cfg)
	if err != nil {
		l := log.Fatal()
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
		for i, e := range etcdEndpoints {
			l.Str(fmt.Sprintf("endpoint-%d", i), e)
		}
		l.Msg("etcd healthcheck failed.")
	}

	log.Info().Msg("etcd connection established.")
	return &APIExecutor {
		kapi,
	}, nil
}

func (e *APIExecutor) Bind(d *APIDispatcher) {
	go e.HandleAgentRegistration(d.RegisterCH)
}

func NewDispatcher(bindAddr string, bindPort uint16) (*APIDispatcher, error) {
	return &APIDispatcher{
		bindAddr,
		bindPort,
		make(chan RegisterRpc, 512),
		make(chan ConnectRequest, 2048),
	}, nil
}

func (api *APIDispatcher) Run() error {
	router := fasthttprouter.New()

	router.POST("/auth", api.AuthHandler)
	router.POST("/register/:domain", api.RegisterHandler())
	router.GET("/connect", api.ConnectHandler())
	log.Debug().Msg("Registered handlers")

	log.Info().
		Str("BindAddr", api.bindAddr).
		Uint16("BindPort", api.bindPort).
		Msg("Starting HTTP API.")

	if err := fasthttp.ListenAndServe(
		fmt.Sprintf("%s:%d", api.bindAddr, api.bindPort),
		router.Handler,
	); err != nil {
		log.Error().
			Str("error", err.Error()).
			Msg("HTTP API exited unexpectedly.")
		return err
	}
	return nil
}