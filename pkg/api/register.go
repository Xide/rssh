package api

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"go.etcd.io/etcd/client"
)

type RegisterRequest struct {
	Host string
}

type registerError struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

type RegisterResponse struct {
	AgentID *string `json:"agentID"`
	Err     *string `json:"error"`
}

func (api *APIExecutor) HandleAgentRegistration(ctx *fasthttp.RequestCtx) *RegisterResponse {
	domain := getDomain(ctx)
	log.Info().Str("domain", domain).Msg("Creating new agent.")
	_, err := api.etcd.Get(context.Background(), "/agents/"+domain, nil)
	if err != nil {
		if err.(client.Error).Code == client.ErrorCodeKeyNotFound {
			log.Debug().Str("domain", domain).Msg("Domain is free.")
			creds, err := GenerateAgentCredentials(domain)
			if err != nil {
				log.Error().
					Str("domain", domain).
					Msg(fmt.Sprintf("Failed to generate agent credentials : %s", err.Error()))
				failRequest(ctx, "Credentials generation error.", 500)
			} else {
				PersistAgentCredentials(api.etcd, *creds)
				return &RegisterResponse{
					AgentID: creds.Secret,
					Err:     nil,
				}
			}
		} else {
			log.Error().
				Str("domain", domain).
				Str("error", err.Error()).
				Msg("Unexpected etcd error")
			failRequest(ctx, "Backend consensus error.", 500)
		}
	} else {
		log.Debug().
			Str("domain", domain).
			Msg("Register for an already occupied slot.")
		failRequest(ctx, "domain already registered.", 403)
	}
	return nil
}

func (api *APIDispatcher) RegisterHandler(ctx *fasthttp.RequestCtx) {
	err := api.executor.HandleAgentRegistration(ctx)
	if err != nil {
		return
	}
	resp := RegisterResponse{nil, nil}
	if err := respond(ctx, resp); err != nil {
		return
	}

	log.Info().
		Str("Domain", getDomain(ctx)).
		Msg("New agent registered.")
}
