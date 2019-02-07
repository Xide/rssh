package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Xide/rssh/pkg/gatekeeper"
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
				log.Error().
					Str("error", err.Error()).
					Msg("Could not unmarshal credentials from etcd.")
				failRequest(ctx, "Inconsistent state for domain", 500)
			} else {
				if persistedCreds.ID.String() == id {
					log.Debug().
						Str("domain", domain).
						Str("agentID", id).
						Msg("Authentication request validated")
					h(ctx)
				} else {
					log.Debug().
						Str("expected", persistedCreds.ID.String()).
						Str("received", id).
						Str("domain", domain).
						Msg("Invalid agent ID.")
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
			gk := resp.Node.Value
			gMeta := &gatekeeper.Meta{}
			err := json.Unmarshal([]byte(gk), gMeta)
			if err != nil {
				log.Warn().Str("error", err.Error()).Msg("Failed to serialize internal gatekeeper state.")
				failRequest(ctx, "Failed to load Gatekeeper state.", 500)
				return
			}
			ctx.SetUserValue("gatekeeper", gMeta)
			h(ctx)
		}
	})
}

// MWithNewSlotFS allocate a slot in an available executor
func MWithNewSlotFS(h fasthttp.RequestHandler, etcd client.KeysAPI) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		log.Debug().Msg("Creating new gatekeeper slot.")
		resp, err := etcd.Get(context.Background(), "/gatekeeper/slotfs", nil)
		if err != nil && err.(client.Error).Code != client.ErrorCodeKeyNotFound {
			failRequest(ctx, "Backend consensus error", 500)
		} else {
			gkMeta := ctx.UserValue("gatekeeper").(*gatekeeper.Meta)
			if resp == nil || resp.Node == nil {
				log.Debug().Msg("Gatekeeper is empty")
				// Pick one slot at random
				ctx.SetUserValue("slot", gkMeta.LowPort)
			} else {
				last := gkMeta.LowPort - 1
				for _, e := range resp.Node.Nodes {
					if e.Dir {
						// Skipping the root directory /gatekeeper/slotfs/
						continue
					}
					sl := strings.Split(e.Key, "/")
					usedPort, err := strconv.ParseUint(sl[len(sl)-1], 10, 16)
					if err != nil {
						log.Error().Str("error", err.Error()).Msg("Failed to parse slotFS entry")
						return
					}
					if uint16(usedPort) != last+1 {
						log.Debug().
							Uint("port", uint(last+1)).
							Msg("Reusing liberated port.")
						break
					}
					last++
				}
				if last >= gkMeta.HighPort {
					log.Warn().
						Msg("All gatekeeper slots already in use.")
					failRequest(ctx, "All gatekeeper slots already in use.", 503)
					return
				}
				ctx.SetUserValue("slot", last+1)
			}

			if _, err = etcd.Set(
				context.Background(),
				fmt.Sprintf("/gatekeeper/slotfs/%d", ctx.UserValue("slot")),
				"{}",
				nil,
			); err != nil {
				failRequest(ctx, "Backend consensus error", 500)
			} else {
				domain, _ := getDomain(ctx)
				log.Info().
					Uint("port", uint(ctx.UserValue("slot").(uint16))).
					Str("domain", domain).
					Msg("Allocated reverse SSH port.")
				h(ctx)
			}
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
