package api

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

type Endpoint struct {
	Host string
	Port uint16
}

func getIdentity(ctx *fasthttp.RequestCtx) string {
	return ctx.UserValue("identity").(string)
}

func getDomain(ctx *fasthttp.RequestCtx) string {
	return ctx.UserValue("domain").(string)
}

func respond(ctx *fasthttp.RequestCtx, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		log.Error().
			Str("error", err.Error()).
			Msg("Failed to marshal response")
		ctx.SetStatusCode(500)
		return err
	}

	if _, err := ctx.Write(b); err != nil {
		log.Warn().
			Str("error", err.Error()).
			Str("response", string(b)).
			Msg("Could not respond to client")
		return err
	}
	return nil
}

func failRequest(ctx *fasthttp.RequestCtx, msg string, code int) {
	ctx.SetStatusCode(code)
	resp := registerError{
		Msg:  msg,
		Code: code,
	}
	respond(ctx, resp)
}
