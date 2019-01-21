package api

import (
	"fmt"
	"errors"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

type AuthRequest struct {
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
	log.Debug().Msg("Received new auth request.")
	token := ctx.QueryArgs().Peek("token")
	req := AuthRequest{
		AgentID: string(token),
	}
	if err := req.Validate(); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		fmt.Fprint(ctx, err.Error())
		return
	}
	log.Debug().
		Str("token", string(token)).
		Msg("Auth request parsed")

	resp := AuthResponse { Slot: nil }
	b, err := json.Marshal(resp)
	if err != nil {
		log.Error().Str("token", string(token)).Msg("Failed to marshal response: " + err.Error())
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	if _, err := ctx.Write(b); err != nil {
		log.Error().
			Str("token", string(token)).
			Str("response", string(b)).
			Str("error", err.Error()).
			Msg("Could not respond to client")
		return
	}
	log.Info().
		Str("token", string(token)).
		Str("response", string(b)).
		Msg("Agent authenticated.")
}
