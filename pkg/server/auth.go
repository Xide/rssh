package api

import (
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

type AuthRequest struct {
	Domain  string
	AgentID string
}

type AuthResponse struct {
	Slot *Endpoint
}

func (r *AuthRequest) Validate() error {
	if len(r.AgentID) == 0 {
		return errors.New("Empty agent id")
	}
	return nil
}

func (api *APIDispatcher) AuthHandler(ctx *fasthttp.RequestCtx) {
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
	resp := AuthResponse{Slot: nil}

	respond(ctx, resp)
	log.Info().
		Str("token", string(token)).
		Str("response", fmt.Sprintf("%v", resp)).
		Msg("Agent authenticated.")
}
