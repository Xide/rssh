package api

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"go.etcd.io/etcd/client"
)

// MValidateDomain is a middleware handling agent registration subdomain validation
func MValidateDomain(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		domain := getDomain(ctx)
		if err := ValidateDomain(domain); err != nil {
			failRequest(ctx, "Invalid domain request.", 400)
			return
		}
		h(ctx)
	})
}

// MValidateAuthenticationRequest is a middleware used to validate the incoming HTTP request
// It will fail if:
//	- The agent identity is invalid
//	- The domain is invalid
//	- The agent is not registered for this domain
func MValidateAuthenticationRequest(h fasthttp.RequestHandler, etcd client.KeysAPI) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		id := getIdentity(ctx)
		if len(id) == 0 {
			failRequest(ctx, "Empty identity", 400)
		}
		_, err := etcd.Get(context.Background(), fmt.Sprintf("/domains/%s/%s", getDomain(ctx), id), nil)
		if err != nil {
			if err.(client.Error).Code == client.ErrorCodeKeyNotFound {
				failRequest(ctx, "Agent is not registered for this domain.", 403)
			}
			return
		}
		h(ctx)
	})
}

func MValidateDomainIsAvailable(h fasthttp.RequestHandler, etcd client.KeysAPI) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		domain := getDomain(ctx)
		_, err := etcd.Get(context.Background(), "/domains/"+domain, nil)
		if err != nil {
			if err.(client.Error).Code == client.ErrorCodeKeyNotFound {
				log.Debug().Str("domain", domain).Msg("Domain is free.")
				h(ctx)
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

	})
}
