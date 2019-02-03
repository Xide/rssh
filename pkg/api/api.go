package api

import (
	"errors"
	"fmt"

	"github.com/buaazp/fasthttprouter"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

type APIDispatcher struct {
	bindAddr   string
	bindPort   uint16
	bindDomain string

	executor *APIExecutor
}

func NewDispatcher(bindAddr string, bindPort uint16, domain string) (*APIDispatcher, error) {
	return &APIDispatcher{
		bindAddr,
		bindPort,
		domain,
		nil,
	}, nil
}

func (api *APIDispatcher) Run(executor *APIExecutor) error {
	if executor == nil {
		return errors.New("Running dispatcher without executor")
	}
	api.executor = executor
	router := fasthttprouter.New()

	router.POST("/auth/:domain", api.executor.ValidateAuthenticationRequest(api.AuthHandler))
	router.POST("/register/:domain", ValidateDomainRequest(api.RegisterHandler))
	router.GET("/connect", api.ConnectHandler())
	log.Debug().Msg("Registered handlers")

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
