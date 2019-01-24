package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"go.etcd.io/etcd/client"
)

type RegisterRequest struct {
	Host string
}

type registerError struct {
	Msg  string `json:"msg"`
	Code int    `json:"code"`
}

type RegisterResponse struct {
	AgentID *string `json:"agentID"`
	Err     *string `json:"error"`
}

func respond(ctx *fasthttp.RequestCtx, v interface{}) error {
	domain := getDomain(ctx)
	b, err := json.Marshal(v)
	if err != nil {
		log.Error().
			Str("error", err.Error()).
			Str("domain", domain).
			Msg("Failed to marshal response")
		ctx.SetStatusCode(500)
		return err
	}

	if _, err := ctx.Write(b); err != nil {
		log.Warn().
			Str("error", err.Error()).
			Str("domain", domain).
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

func ValidateDomain(domain string) error {
	if match, _ := regexp.MatchString("^[a-zA-Z0-9]+$", domain); !match {
		return errors.New("Illegal characters in requested domain.")
	}
	return nil
}

func getDomain(ctx *fasthttp.RequestCtx) string {
	return ctx.UserValue("domain").(string)
}

func (api *APIExecutor) HandleAgentRegistration(ctx *fasthttp.RequestCtx) *RegisterResponse {
	domain := getDomain(ctx)
	log.Info().Str("domain", domain).Msg("Creating new agent.")
	_, err := api.etcd.Get(context.Background(), "/agents/"+domain, nil)
	if err != nil {
		if err.(client.Error).Code == client.ErrorCodeKeyNotFound {
			log.Debug().Str("domain", domain).Msg("Domain is free.")
			creds, err := GenerateAgentCredentials(domain)
			if err != nil {
				log.Error().
					Str("domain", domain).
					Msg(fmt.Sprintf("Failed to generate agent credentials : %s", err.Error()))
				failRequest(ctx, "Credentials generation error.", 500)
			} else {
				PersistAgentCredentials(api.etcd, *creds, domain)
				return &RegisterResponse{
					AgentID: creds.Secret,
					Err:     nil,
				}
			}
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
	return nil
}

func (api *APIDispatcher) RegisterHandler(ctx *fasthttp.RequestCtx) {
	reqDomain := ctx.UserValue("domain").(string)

	if err := ValidateDomain(reqDomain); err != nil {
		failRequest(ctx, "Invalid domain request.", 400)
		return
	}

	resp := api.executor.HandleAgentRegistration(ctx)
	if resp == nil {
		return
	}

	if err := respond(ctx, resp); err != nil {
		return
	}

	log.Info().
		Str("Domain", reqDomain).
		Msg("New agent registered.")

}
