package gatekeeper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.etcd.io/etcd/client"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog/log"

	"github.com/Xide/rssh/pkg/utils"
)

// Gate represent the metadatas associated with an
// agent persistent connection to the gatekeeper.
type Gate struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

// Meta exposes informations about the gatekeeper runtime
// configuration. This structure will get persisted into etcd at /meta/gatekeeper
type Meta struct {
	SSHAddr  string
	SSHPort  uint16
	LowPort  uint16
	HighPort uint16
}

// GateKeeper is the public SSH server exposing the forwarded agents.
type GateKeeper struct {
	Meta     Meta
	srv      *ssh.Server
	etcd     *client.KeysAPI
	backends []Gate
	clients  []AgentSlot
}

// WithEtcdE instanciate an etcd client and connect to the cluster.
// the resulting api keys are persisted in the GateKeeper.
func (g *GateKeeper) WithEtcdE(etcdEndpoints []string) error {
	k, err := utils.GetEtcdKey(etcdEndpoints)
	if err != nil {
		return err
	}
	g.etcd = k
	// Clear any potential remaining datas from previous gatekeepers
	// WILL prevent multiple gatekeepers to run at the same time.
	_, err = (*k).Delete(context.Background(), "/gatekeeper/slotfs", &client.DeleteOptions{Recursive: true})
	return nil
}

// WithPortRange sets the range on which the gatekeeper will try to
// allocate slots for reverse port forwarding.
func (g *GateKeeper) WithPortRange(low uint16, high uint16) *GateKeeper {
	g.Meta.LowPort = utils.Min(low, high)
	g.Meta.HighPort = utils.Max(low, high)
	return g
}

// Run is the entrypoint of the Gatekeeper.
// it announce itself to etcd and then starts the SSH server.
func (g *GateKeeper) Run() error {
	err := g.announce()
	if err != nil {
		return err
	}
	return g.initSSHServer()
}

// announce persists the Meta structure in etcd.
func (g *GateKeeper) announce() error {
	m, err := json.Marshal(g.Meta)
	if err != nil {
		return err
	}

	log.Debug().Msg("Starting to announce API to etcd")
	_, err = (*g.etcd).Set(context.Background(), "/meta/gatekeeper", string(m), nil)
	if err != nil {
		return err
	}

	log.Info().Msg("Gatekeeper registered in etcd.")
	return nil
}

// NewGateKeeper is the constructor for an empty gatekeeper.
func NewGateKeeper(addr string, port uint16) (*GateKeeper, error) {
	return &GateKeeper{
		srv: nil,
		Meta: Meta{
			SSHAddr:  addr,
			SSHPort:  port,
			LowPort:  30000,
			HighPort: 40000,
		},
	}, nil
}

// collectClosedSession removes from the runtime a connection that has been
// closed by the agent.
func (g *GateKeeper) collectClosedSession(ctx ssh.Context, slot *AgentSlot) {
	payload, err := json.Marshal(slot)
	if err != nil {
		log.Warn().
			Str("error", err.Error()).
			Msg("Failed to marshal slot for etcd garbage collection.")
		return
	}
	<-ctx.Done()

	if _, err := (*g.etcd).Delete(
		context.Background(),
		fmt.Sprintf("/gatekeeper/slotfs/%d", slot.Port),
		&client.DeleteOptions{
			PrevValue: string(payload),
		},
	); err != nil {
		log.Warn().
			Str("error", err.Error()).
			Str("domain", slot.Domain).
			Msg("Could not find slot for garbage collection.")
	} else {
		log.Debug().Str("domain", slot.Domain).Msg("Closed agent connection")

	}
}

// initSSHServer creates a new SSH server with
// - routing logic through command
// - reverse port forwarding logic for agents
func (g *GateKeeper) initSSHServer() error {
	if g.srv != nil {
		return errors.New("SSH server already initialized")
	}
	addr := fmt.Sprintf("%s:%d", g.Meta.SSHAddr, g.Meta.SSHPort)
	server := ssh.Server{
		Addr:    addr,
		Handler: ssh.Handler(g.proxyCommandHandler()),
		ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(g.reversePortForwardHandler(*g.etcd)),
	}
	g.srv = &server
	log.Info().
		Str("addr", g.Meta.SSHAddr).
		Uint16("port", g.Meta.SSHPort).
		Msg("starting SSH server")
	// go g.monitorSlots()
	return server.ListenAndServe()
}
