package api

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"go.etcd.io/etcd/client"
)

// RegisterRequest is the parsed struct representing
// an HTTP request on /register
type RegisterRequest struct {
	Host string
}

// registerError serialize a registration error in the JSON response
type registerError struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

// RegisterResponse serialize a registration response.
type RegisterResponse struct {
	AgentID *AgentCredentials `json:"agentID"`
	Err     *registerError    `json:"error"`
}

// MWithNewAgentCredentials is a middleware that inject new agent credentials in the
// context. The generated credentials can be accessed using `ctx.UserValue("credentials")`.
// MWithNewAgentCredentials will fail with a 500 error code if there is an issue with the
// credentials generation or the etcd comunication.
func MWithNewAgentCredentials(h fasthttp.RequestHandler, etcd client.KeysAPI) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		domain := getDomain(ctx)
		creds, err := GenerateAgentCredentials()

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

// MWithDomainLease is a middleware ensuring that the domain provided by an agent can
// be allocated. If so, it will be allocated in the etcd and passed to subsequent handler.
// Otherwise, it will return an HTTP 500 error.
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

// registerHandlerWrapped serialize the generated agent credentials and return
// them via JSON in the response body.
func (api *Dispatcher) registerHandlerWrapped(ctx *fasthttp.RequestCtx) {
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
// requests down to Dispatcher.registerHandlerWrapper
func (api *Dispatcher) RegisterHandler(ctx *fasthttp.RequestCtx) {
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
