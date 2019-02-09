package gatekeeper

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog/log"

	"github.com/Xide/rssh/pkg/utils"
)

func (g *GateKeeper) getSlotForDomain(domain string) (*AgentSlot, error) {
	return g.getFirstSlotForFn(func(sl *AgentSlot) bool {
		return strings.Compare(sl.Domain, domain) == 0
	})
}

func (g *GateKeeper) setupForward(s ssh.Session, slot *AgentSlot) {
	// 127.0.0.1 is assumed here as we can only have one
	// active gatekeeper at the same time.
	backendAddr := fmt.Sprintf("127.0.0.1:%d", slot.Port)
	conn, err := net.Dial("tcp", backendAddr)
	if err != nil {
		log.Warn().
			Str("domain", slot.Domain).
			Str("destination", backendAddr).
			Str("error", err.Error()).
			Msg("Failed to dial backend.")
	}
	log.Debug().
		Str("domain", slot.Domain).
		Str("destination", backendAddr).
		Msg("Connected to backend, starting forward.")
	go func() {
		defer s.Close()
		defer conn.Close()
		io.Copy(s, conn)
		log.Debug().
			Str("domain", slot.Domain).
			Str("destination", backendAddr).
			Msg("Agent side socket interrupted")
	}()
	go func() {
		defer s.Close()
		defer conn.Close()
		io.Copy(conn, s)
		log.Debug().
			Str("domain", slot.Domain).
			Str("destination", backendAddr).
			Msg("Client side socket interrupted")
	}()
	select {
	case <-s.Context().Done():
		log.Debug().
			Str("domain", slot.Domain).
			Str("destination", backendAddr).
			Msg("Proxy command finished.")
	}
}

func parseRequestedDomain(s ssh.Session) (string, error) {
	if len(s.Command()) > 0 {
		destDomain := s.Command()[0]
		return destDomain, nil
	}
	log.Debug().Msg("Unsupported connection request.")
	return "", errors.New("proxy request without command")
}

func (g *GateKeeper) proxyCommandHandler() func(ssh.Session) {
	return func(s ssh.Session) {
		destDomain, err := parseRequestedDomain(s)
		subDomain, _ := utils.SplitDomainRequest(destDomain)
		if err != nil {
			io.WriteString(s, fmt.Sprintf("Unsupported connection request."))
			return
		}
		log.Debug().Str("domain", destDomain).Msg("Client requested proxy")
		slot, err := g.getSlotForDomain(subDomain)
		if err != nil {
			log.Warn().
				Str("error", err.Error()).
				Str("domain", subDomain).
				Msg("Domain not found")
			io.WriteString(s, fmt.Sprintf("Domain %s not found.", destDomain))
		} else {
			g.setupForward(s, slot)
		}
	}
}
