package gatekeeper

import (
	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog/log"
)

func (g *GateKeeper) reversePortForwardHandler() func(ssh.Context, string, uint32) bool {
	return func(ctx ssh.Context, host string, port uint32) bool {
		log.Debug().
			Str("client_addr", host).
			Uint32("port", port).
			Msg("Port forward request")
		for x := 0; x < len(g.clients); x++ {
			if uint16(port) == g.clients[x].Port {
				if g.clients[x].Established {
					log.Debug().
						Str("client_addr", host).
						Uint32("port", port).
						Msg("A session is already established for this agent.")
					return false
				}
				g.clients[x].Established = true
				go g.collectClosedSession(ctx, &g.clients[x])
				log.Debug().
					Str("client_addr", host).
					Uint32("port", port).
					Msg("Accepted port forward")
				return true
			}
		}

		log.Debug().
			Str("client_addr", host).
			Uint32("port", port).
			Str("error", "slot not found").
			Msg("Denied port forward.")
		return false
	}
}
