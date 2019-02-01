package api

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/valyala/fasthttp"
	"go.etcd.io/etcd/client"
)

// ValidateDomain returns an error if the parameter is not a valid subdomain
// We only allow alphanumeric characters
func ValidateDomain(domain string) error {
	if match, _ := regexp.MatchString("^[a-zA-Z0-9]+$", domain); !match {
		return errors.New("illegal characters in requested domain")
	}
	return nil
}

// ValidateDomainRequest is a middleware handling agent registration subdomain validation
func ValidateDomainRequest(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		domain := getDomain(ctx)
		if err := ValidateDomain(domain); err != nil {
			failRequest(ctx, "Invalid domain request.", 400)
			return
		}
		h(ctx)
	})
}

// ValidateAuthenticationRequest is a middleware used to validate the incoming HTTP request
// It will fail if:
//	- The agent identity is invalid
//	- The domain is invalid
//	- The agent is not registered for this domain
func (api *APIExecutor) ValidateAuthenticationRequest(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return ValidateDomainRequest(fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		id := getIdentity(ctx)
		if len(id) == 0 {
			failRequest(ctx, "Empty identity", 400)
		}
		_, err := api.etcd.Get(context.Background(), fmt.Sprintf("/domains/%s/%s", getDomain(ctx), id), nil)
		if err != nil {
			if err.(client.Error).Code == client.ErrorCodeKeyNotFound {
				failRequest(ctx, "Agent is not registered for this domain.", 403)
			}
			return
		}
		h(ctx)
	}))
}
