package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

// NewDispatcher is a simple wrapper to construct a Dispatcher structure
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

// HealthHandler will respond to GET /health request in order for
// clients / agents / orchestrators to know if this service can
// handle traffic.
// TODO: check for ETCD connectivity
func (api *Dispatcher) HealthHandler(ctx *fasthttp.RequestCtx) {
	payload, err := json.Marshal(struct {
		Ok   bool   `json:"ok"`
		Time string `json:"time"`
	}{
		true,
		time.Now().String(),
	})
	if err != nil {
		ctx.SetStatusCode(500)
		log.Warn().
			Str("error", err.Error()).
			Msg("Failed to serialize healthcheck infos.")
	} else {
		ctx.SetStatusCode(200)
		_, err = ctx.Write(payload)
		if err != nil {
			log.Warn().
				Str("error", err.Error()).
				Msg("Failed to respond to healthcheck.")
		}
	}
}

// Run is the entry point of the dispatcher.
// it does the following:
// - Connect to etcd
// - Create the HTTP routes
// - Listen and serve requests
func (api *Dispatcher) Run() error {

	var k *client.KeysAPI
	if err := utils.WithFixedIntervalRetry(
		func() error {
			var err error
			k, err = utils.GetEtcdKey(api.etcdEndpoints)
			return err
		},
		5,
		5*time.Second,
	); err != nil {
		return err
	}

	api.etcd = k
	err := api.announce()
	if err != nil {
		return err
	}
	router := fasthttprouter.New()

	router.GET("/health", api.HealthHandler)
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
