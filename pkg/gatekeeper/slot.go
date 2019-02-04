package gatekeeper

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
)

// AgentSlot represents a pending or active authorized connection
// to the GateKeeper.
type AgentSlot struct {
	// Subdomain on which the agent is registered
	Domain      string `json:"domain"`
	Port        uint16 `json:"port"`
	AgentID     string `json:"agentID"`
	Established bool   `json:"established"`
}

func (g *GateKeeper) allocateAgentSlot(domain string) (*AgentSlot, error) {
	for x := 0; x < len(g.clients); x++ {
		if strings.Compare(domain, g.clients[x].Domain) == 0 {
			log.Warn().
				Str("domain", domain).
				Uint16("bound_port", g.clients[x].Port).
				Msg("Domain already registered for another agent")
			return nil, errors.New("domain already registered")
		}
	}

	port, err := findAvailablePort(g.Meta.LowPort, g.Meta.HighPort)
	if err != nil {
		return nil, err
	}

	slot := &AgentSlot{
		Domain:      domain,
		Port:        port,
		AgentID:     "test",
		Established: false,
	}
	g.clients = append(g.clients, *slot)
	return slot, nil
}
