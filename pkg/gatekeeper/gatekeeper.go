package gatekeeper

import (
	"errors"
	"fmt"
	"strings"

	"go.etcd.io/etcd/client"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog/log"

	"github.com/Xide/rssh/pkg/utils"
)

type Gate struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

type GateKeeper struct {
	srv       *ssh.Server
	etcd      *client.KeysAPI
	frontGate Gate
	backends  []Gate
	clients   []AgentSlot
	lowPort   uint16
	highPort  uint16
}

func (g *GateKeeper) WithEtcdE(etcdEndpoints []string) error {
	k, err := utils.GetEtcdKey(etcdEndpoints)
	if err != nil {
		return err
	}
	g.etcd = k
	return nil
}

func (g *GateKeeper) WithPortRange(low uint16, high uint16) *GateKeeper {
	g.lowPort = utils.Min(low, high)
	g.highPort = utils.Max(low, high)
	return g
}

func NewGateKeeper(addr string, port uint16) (*GateKeeper, error) {
	return &GateKeeper{
		srv:       nil,
		frontGate: Gate{Host: addr, Port: port},
		lowPort:   30000,
		highPort:  40000,
	}, nil
}

func (g *GateKeeper) collectClosedSession(ctx ssh.Context, slot *AgentSlot) {
	<-ctx.Done()
	for x := 0; x < len(g.clients); x++ {
		if strings.Compare(slot.Domain, g.clients[x].Domain) == 0 {
			g.clients = append(g.clients[:x], g.clients[x+1:]...)
			log.Debug().Str("domain", slot.Domain).Msg("Closed agent connection")
			return
		}
	}
	log.Warn().Str("domain", slot.Domain).Msg("Could not find slot for garbage collection.")
}

func (g *GateKeeper) InitSSHServer() error {
	if g.srv != nil {
		return errors.New("SSH server already initialized")
	}
	addr := fmt.Sprintf("%s:%d", g.frontGate.Host, g.frontGate.Port)
	server := ssh.Server{
		Addr:    addr,
		Handler: ssh.Handler(g.proxyCommandHandler()),
		ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(g.reversePortForwardHandler()),
	}
	g.srv = &server
	log.Info().
		Str("addr", g.frontGate.Host).
		Uint16("port", g.frontGate.Port).
		Msg("starting SSH server")
	return server.ListenAndServe()
}
