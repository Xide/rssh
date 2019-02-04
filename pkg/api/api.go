package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Xide/rssh/pkg/utils"
	"github.com/buaazp/fasthttprouter"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"go.etcd.io/etcd/client"
)

// Meta represents metadatas about the running api.
// It will be persisted to etcd in order to configure
// the gatekeepers.
type Meta struct {
	BindDomain string `json:"domain"`
	BindAddr   string `json:"addr"`
	BindPort   uint16 `json:"port"`
}

// Dispatcher is the API entry point, it will compose the middlewares
// flow and pass them the request
type Dispatcher struct {
	Meta Meta

	etcdEndpoints []string
	etcd          *client.KeysAPI
}

func NewDispatcher(
	bindAddr string,
	bindPort uint16,
	domain string,
	etcdEndpoints []string,
) (*Dispatcher, error) {
	return &Dispatcher{
		Meta{
			domain,
			bindAddr,
			bindPort,
		},
		etcdEndpoints,
		nil,
	}, nil
}

// announce write the current parameters and Metadatas to the etcd cluster.
func (api *Dispatcher) announce() error {
	m, err := json.Marshal(api.Meta)
	if err != nil {
		return err
	}

	log.Debug().Msg("Starting to announce API to etcd")
	_, err = (*api.etcd).Set(context.Background(), "/meta/api", string(m), nil)
	if err != nil {
		return err
	}

	log.Info().Msg("API registered in etcd.")
	return nil
}

func (api *Dispatcher) Run() error {

	kapi, err := utils.GetEtcdKey(api.etcdEndpoints)
	if err != nil {
		return err
	}
	api.etcd = kapi
	err = api.announce()
	if err != nil {
		return err
	}
	router := fasthttprouter.New()

	router.POST("/auth/:domain", api.AuthHandler)
	router.POST("/register/:domain", MValidateDomain(api.RegisterHandler))

	log.Info().
		Str("domain", api.Meta.BindDomain).
		Str("BindAddr", api.Meta.BindAddr).
		Uint16("BindPort", api.Meta.BindPort).
		Msg("Starting HTTP API.")

	if err := fasthttp.ListenAndServe(
		fmt.Sprintf("%s:%d", api.Meta.BindAddr, api.Meta.BindPort),
		router.Handler,
	); err != nil {
		log.Error().
			Str("error", err.Error()).
			Msg("HTTP API exited unexpectedly.")
		return err
	}
	return nil
}
