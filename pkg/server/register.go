package api

import (
	"fmt"
	"regexp"
	"errors"
	"context"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
	"go.etcd.io/etcd/client"
)

type RegisterRequest struct {
	Host string
}

type response struct {
	AgentID string
}

type RegisterResponse struct {
	Err error
	Data *response
}

type RegisterRpc struct {
	Resp chan RegisterResponse
	Req  RegisterRequest
}

func (req *RegisterRequest) Validate() error {
	if match, _ := regexp.MatchString("^[a-zA-Z0-9]+$", req.Host); !match {
		return errors.New("Illegal characters in requested domain.")
	}
	return nil
}


func (api *APIExecutor) HandleAgentRegistration(reqCH chan RegisterRpc) {
	for {
		rpc := <- reqCH

		go func() {
			log.Info().Str("domain", rpc.Req.Host).Msg("Creating new agent.")
			_, err := api.etcd.Get(context.Background(), "/agents/" + rpc.Req.Host, nil)
			if err != nil {
				if err.(client.Error).Code == client.ErrorCodeKeyNotFound {
					log.Debug().Str("domain", rpc.Req.Host).Msg("Domain is free.")
					_, err = GenerateAgentCredentials(rpc.Req.Host)
					if err != nil {
						log.Error().
							Str("domain", rpc.Req.Host).
							Msg(fmt.Sprintf("Failed to generate agent credentials : %s", err.Error()))
						rpc.Resp <- RegisterResponse {
							Err: errors.New("Credentials generation error."),
							Data: nil,
						}
						return
					}

					rpc.Resp <- RegisterResponse{
						Err: nil,
						Data: nil, 
					}

				} else {
					log.Error().
						Str("domain", rpc.Req.Host).
						Str("error", err.Error()).
						Msg("Unexpected etcd error")
					rpc.Resp <- RegisterResponse {
						Err: errors.New("Backend consensus error."),
						Data: nil,
					}
				}
			} else {
				log.Debug().
					Str("domain", rpc.Req.Host).
					Msg("Register for an already occupied slot.")
				rpc.Resp <- RegisterResponse{
					Err: errors.New("domain already registered."),
					Data: nil, 
				}
			}
		}()
	}
} 

func (api *APIDispatcher) RegisterHandler() (func(ctx *fasthttp.RequestCtx)){
	return func(ctx *fasthttp.RequestCtx) {
		reqDomain := ctx.UserValue("domain").(string)
		req := RegisterRpc{
			Resp: make(chan RegisterResponse, 1),
			Req: RegisterRequest {Host: reqDomain},
		}

		if err := req.Req.Validate(); err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			fmt.Fprint(ctx, err.Error())
			return
		}

		api.RegisterCH <- req
		resp := <- req.Resp

		b, err := json.Marshal(resp)
		if err != nil {
			log.Error().Str("domain", reqDomain).Msg("Failed to marshal response: " + err.Error())
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			return
		}
		if _, err := ctx.Write(b); err != nil {
			log.Error().
				Str("domain", reqDomain).
				Str("response", string(b)).
				Str("error", err.Error()).
				Msg("Could not respond to client")
			return
		}

		log.Info().
			Str("Domain", reqDomain).
			Msg("New agent registered.")
	
	}
}
