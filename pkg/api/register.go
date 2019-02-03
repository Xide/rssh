package api

import (
	"context"
	"encoding/json"

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
	AgentID *AgentCredentials `json:"agentID"`
	Err     *string           `json:"error"`
}

func MWithNewAgentCredentials(h fasthttp.RequestHandler, etcd client.KeysAPI) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		domain := getDomain(ctx)
		creds, err := GenerateAgentCredentials(domain)

		if err != nil {
			log.Error().
				Str("error", err.Error()).
				Str("domain", domain).
				Msg("Failed to generate agent credentials")
			failRequest(ctx, "Credentials generation error.", 500)
		} else {
			err = PersistAgentCredentials(etcd, *creds)
			if err != nil {
				log.Error().
					Str("error", err.Error()).
					Str("domain", domain).
					Msg("Could not persist agent credentials")
				failRequest(ctx, "Credentials generation error.", 500)
			} else {
				ctx.SetUserValue("credentials", creds)
				h(ctx)
			}
		}
	})
}

func MWithDomainLease(h fasthttp.RequestHandler, etcd client.KeysAPI) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		domain := getDomain(ctx)
		credentials := *ctx.UserValue("credentials").(*AgentCredentials)
		credentials.DropSecrets()
		options := client.SetOptions{
			PrevExist: client.PrevNoExist,
		}
		m, err := json.Marshal(credentials)
		if err != nil {
			log.Error().
				Str("error", err.Error()).
				Str("agent", credentials.ID.String()).
				Str("domain", domain).
				Msg("Could not serialize credentials")
			failRequest(ctx, "Domain allocation error.", 500)
		} else {
			_, err = etcd.Set(
				context.Background(),
				"/domains/"+domain,
				string(m),
				&options,
			)
			if err != nil {
				log.Error().
					Str("error", err.Error()).
					Str("agent", credentials.ID.String()).
					Str("domain", domain).
					Msg("Could not allocate domain")
				failRequest(ctx, "Domain allocation error.", 500)
			} else {
				log.Info().
					Str("agent", credentials.ID.String()).
					Str("domain", domain).
					Msg("Allocated domain")
				h(ctx)
			}
		}
	})
}

func (api *APIDispatcher) registerHandlerWrapped(ctx *fasthttp.RequestCtx) {
	resp := RegisterResponse{
		AgentID: ctx.UserValue("credentials").(*AgentCredentials),
		Err:     nil,
	}
	if err := respond(ctx, resp); err != nil {
		return
	}

	log.Info().
		Str("Domain", getDomain(ctx)).
		Msg("New agent registered.")
}

// RegisterHandler is the entrypoint for an HTTP POST request in the API
// It uses several middlewares to validate requests, and pass the valid
// requests down to APIDispatcher.registerHandlerWrapper
func (api *APIDispatcher) RegisterHandler(ctx *fasthttp.RequestCtx) {
	MValidateDomain(
		MValidateDomainIsAvailable(
			MWithNewAgentCredentials(
				MWithDomainLease(
					api.registerHandlerWrapped,
					*api.etcd,
				),
				*api.etcd,
			),
			*api.etcd,
		),
	)(ctx)
}
