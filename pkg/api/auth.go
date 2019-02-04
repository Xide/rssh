package api

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

// AuthRequest describe the content of an authentication request
type AuthRequest struct {
	// domain the agent is attempting to bind to
	Domain string
	// UUID returned by the register API call
	AgentID string
}

// AuthResponse describe the content of the authentication response
type AuthResponse struct {
	Port uint16 `json:"port"`
}

// Validate return an error if the agent id is invalid.
func (r *AuthRequest) Validate() error {
	if len(r.AgentID) == 0 {
		return errors.New("Empty agent id")
	}
	return nil
}

// authHandlerWrapped is called at the sink of the middleware chain.
func (api *Dispatcher) authHandlerWrapped(ctx *fasthttp.RequestCtx) {
	log.Debug().Str("domain", getDomain(ctx)).Msg("Received new auth request.")
	token := getIdentity(ctx)
	req := AuthRequest{
		AgentID: string(token),
		Domain:  getDomain(ctx),
	}
	if err := req.Validate(); err != nil {
		failRequest(ctx, err.Error(), 400)
		return
	}
	log.Debug().
		Str("token", string(token)).
		Msg("Auth request parsed")

	// Create an available slot for the agent to connect to.
	resp := AuthResponse{Port: 0}

	respond(ctx, resp)
	log.Info().
		Str("token", string(token)).
		Str("response", fmt.Sprintf("%v", resp)).
		Msg("Agent authenticated.")
}

// AuthHandler is the entrypoint for an HTTP POST request in the API
// It uses several middlewares to validate requests, and pass the valid
// requests down to Dispatcher.registerHandlerWrapper
func (api *Dispatcher) AuthHandler(ctx *fasthttp.RequestCtx) {
	MValidateDomain(
		MValidateAuthenticationRequest(
			api.authHandlerWrapped,
			*api.etcd,
		),
	)(ctx)
}
