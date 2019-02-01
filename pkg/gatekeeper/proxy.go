package gatekeeper

import (
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog/log"
)

func (g *GateKeeper) proxyCommandHandler() func(ssh.Session) {
	return func(s ssh.Session) {
		if len(s.Command()) > 0 {
			destDomain := s.Command()[0]
			log.Debug().Str("domain", destDomain).Msg("Client requested proxy")
			for x := 0; x < len(g.clients); x++ {
				if strings.Compare(destDomain, g.clients[x].Domain) == 0 {
					backendAddr := fmt.Sprintf("%s:%d", g.clients[x].Host, g.clients[x].Port)
					conn, err := net.Dial("tcp", backendAddr)
					if err != nil {
						log.Warn().
							Str("domain", destDomain).
							Str("destination", backendAddr).
							Str("error", err.Error()).
							Msg("Failed to dial backend.")
					}
					log.Debug().
						Str("domain", destDomain).
						Str("destination", backendAddr).
						Msg("Connected to backend, starting forward.")
					go func() {
						io.Copy(s, conn)
						log.Debug().
							Str("domain", destDomain).
							Str("destination", backendAddr).
							Msg("Agent side socket interrupted")
					}()
					go func() {
						io.Copy(conn, s)
						log.Debug().
							Str("domain", destDomain).
							Str("destination", backendAddr).
							Msg("Client side socket interrupted")
					}()
					select {
					case <-s.Context().Done():
						log.Debug().
							Str("domain", destDomain).
							Str("destination", backendAddr).
							Msg("Proxy command finished.")

					}
					return
				}
			}
			log.Warn().
				Str("domain", destDomain).
				Msg("Domain not found")

			io.WriteString(s, fmt.Sprintf("Domain %s not found.", destDomain))
		} else {
			log.Debug().Msg("Unsupported connection request.")
			io.WriteString(s, fmt.Sprintf("Unsupported connection request."))
		}
	}
}
