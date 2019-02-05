package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"go.etcd.io/etcd/client"
)

// MValidateDomain is a middleware handling agent registration subdomain validation
func MValidateDomain(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		domain, err := getDomain(ctx)
		if err != nil || ValidateDomain(domain) != nil {
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
		id, err := getIdentity(ctx)
		if err != nil || len(id) == 0 {
			failRequest(ctx, "Empty identity", 400)
			return
		}
		domain, _ := getDomain(ctx)
		resp, err := etcd.Get(context.Background(), fmt.Sprintf("/domains/%s", domain), nil)
		if err != nil {
			if err.(client.Error).Code == client.ErrorCodeKeyNotFound {
				failRequest(ctx, "Agent is not registered for this domain.", 403)
			} else {
				failRequest(ctx, "Backend consensus error.", 500)
			}
		} else {
			persistedCreds := AgentCredentials{}
			err = json.Unmarshal([]byte(resp.Node.Value), &persistedCreds)
			if err != nil {
				fmt.Println(resp.Node.Value)
				fmt.Println(err.Error())
				failRequest(ctx, "Inconsistent state for domain", 500)
			} else {
				if persistedCreds.ID.String() == id {
					log.Debug().
						Str("domain", domain).
						Str("agentID", id).
						Msg("Authentication request validated")
					h(ctx)
				} else {
					failRequest(ctx, "Invalid agent ID for this domain.", 403)
				}
			}

		}
	})
}

// MWithGatekeeperMeta injects the value of the etcd `/meta/gatekeeper` key into the context.
// It will fail and return a 500 error code if the informations can't be extracted from etcd.
func MWithGatekeeperMeta(h fasthttp.RequestHandler, etcd client.KeysAPI) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		resp, err := etcd.Get(context.Background(), "/meta/gatekeeper", nil)
		if err != nil {
			if err.(client.Error).Code == client.ErrorCodeKeyNotFound {
				failRequest(ctx, "Gatekeeper is not available", 500)
			} else {
				failRequest(ctx, "Backend consensus error", 500)
			}
		} else {
			ctx.SetUserValue("gatekeeper", resp.Node.Value)
			h(ctx)
		}
	})
}

// MValidateDomainIsAvailable will check for the presence of the domain in etcd. It will only
// pass requests to the subsequent handler if the domain isn't already reserved. Otherwise,
// it can yield 403 code for an already registered domain or 500 in case of an etcd failure.
func MValidateDomainIsAvailable(h fasthttp.RequestHandler, etcd client.KeysAPI) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		domain, _ := getDomain(ctx)
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
