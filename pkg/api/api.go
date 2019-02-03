package api

import (
	"fmt"

	"github.com/Xide/rssh/pkg/utils"
	"github.com/buaazp/fasthttprouter"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"go.etcd.io/etcd/client"
)

type APIDispatcher struct {
	bindAddr   string
	bindPort   uint16
	bindDomain string

	etcdEndpoints []string
	etcd          *client.KeysAPI
}

func NewDispatcher(
	bindAddr string,
	bindPort uint16,
	domain string,
	etcdEndpoints []string,
) (*APIDispatcher, error) {
	return &APIDispatcher{
		bindAddr,
		bindPort,
		domain,
		etcdEndpoints,
		nil,
	}, nil
}

func (api *APIDispatcher) Run() error {

	kapi, err := utils.GetEtcdKey(api.etcdEndpoints)
	if err != nil {
		return err
	}
	api.etcd = kapi
	router := fasthttprouter.New()

	router.POST("/auth/:domain", api.AuthHandler)
	router.POST("/register/:domain", MValidateDomain(api.RegisterHandler))

	log.Info().
		Str("domain", api.bindDomain).
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
