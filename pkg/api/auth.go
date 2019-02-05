package api

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"

	"github.com/Xide/rssh/pkg/gatekeeper"
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

	// Ignoring errors here as it already been checked by the middlewares.
	domain, _ := getDomain(ctx)
	token, _ := getIdentity(ctx)

	log.Debug().Str("domain", domain).Msg("Received new auth request.")

	req := AuthRequest{
		AgentID: string(token),
		Domain:  domain,
	}
	if err := req.Validate(); err != nil {
		log.Debug().Str("error", err.Error()).Msg("Failed to validate auth request.")
		failRequest(ctx, err.Error(), 400)
		return
	}
	log.Debug().
		Str("token", string(token)).
		Msg("Auth request parsed")

	// Get Gatekeeper port
	gk := ctx.UserValue("gatekeeper").(string)
	gMeta := &gatekeeper.Meta{}
	err := json.Unmarshal([]byte(gk), gMeta)
	if err != nil {
		log.Warn().Str("error", err.Error()).Msg("Failed to serialize internal gatekeeper state.")
		failRequest(ctx, "Failed to load Gatekeeper state.", 500)
		return
	}
	// Create an available slot for the agent to connect to.
	resp := AuthResponse{Port: gMeta.SSHPort}

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
			MWithGatekeeperMeta(
				api.authHandlerWrapped,
				*api.etcd,
			),
			*api.etcd,
		),
	)(ctx)
}
