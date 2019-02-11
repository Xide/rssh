package api

import (
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

// GkConnectInfos describe the content of the authentication response
type GkConnectInfos struct {
	GkMeta gatekeeper.Meta `json:"gk"`
	Port   uint16          `json:"port"`
}

// AuthResponse describe the contents of the HTTP response
type AuthResponse struct {
	Infos *GkConnectInfos `json:"connection"`
	Err   *Error          `json:"error"`
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

	// Get Gatekeeper port
	gMeta := ctx.UserValue("gatekeeper").(*gatekeeper.Meta)
	// Create an available slot for the agent to connect to.
	// TODO: better method than random
	resp := AuthResponse{
		Infos: &GkConnectInfos{
			Port:   ctx.UserValue("slot").(uint16),
			GkMeta: *gMeta,
		},
		Err: nil,
	}

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
				MWithNewSlotFS(
					api.authHandlerWrapped,
					*api.etcd,
				),
				*api.etcd,
			),
			*api.etcd,
		),
	)(ctx)
}
