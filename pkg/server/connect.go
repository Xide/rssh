package api

import (
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

type ConnectRequest struct {
	Host string
}

type ConnectResponse struct{}

func (api *APIDispatcher) ConnectHandler() func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		log.Debug().
			Str("Host", string(ctx.Host())).
			Msg("Connection request.")
		respond(ctx, ConnectResponse{})
	}
}
