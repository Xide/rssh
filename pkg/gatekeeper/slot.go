package gatekeeper

import (
	"errors"
	"strings"

	"github.com/rs/zerolog/log"
)

type AgentSlot struct {
	Domain      string `json:"domain"`
	Host        string `json:"host"`
	Port        uint16 `json:"port"`
	AgentID     string `json:"agentID"`
	Established bool   `json:"established"`
}

func (g *GateKeeper) AllocateAgentSlot(domain string) (*AgentSlot, error) {
	for x := 0; x < len(g.clients); x++ {
		if strings.Compare(domain, g.clients[x].Domain) == 0 {
			log.Warn().
				Str("domain", domain).
				Uint16("bound_port", g.clients[x].Port).
				Msg("Domain already registered for another agent")
			return nil, errors.New("domain already registered")
		}
	}

	port, err := findAvailablePort(g.lowPort, g.highPort)
	if err != nil {
		return nil, err
	}

	slot := &AgentSlot{
		Domain:      domain,
		Host:        "127.0.0.1",
		Port:        port,
		AgentID:     "test",
		Established: false,
	}
	g.clients = append(g.clients, *slot)
	return slot, nil
}
