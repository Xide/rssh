package gatekeeper

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"go.etcd.io/etcd/client"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog/log"
	gossh "golang.org/x/crypto/ssh"

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
	hostKey  gossh.Signer
}

// WithEtcdE instanciate an etcd client and connect to the cluster.
// the resulting api keys are persisted in the GateKeeper.
func (g *GateKeeper) WithEtcdE(etcdEndpoints []string) error {
	var k *client.KeysAPI
	if err := utils.WithFixedIntervalRetry(
		func() error {
			var err error
			k, err = utils.GetEtcdKey(etcdEndpoints)
			return err
		},
		5,
		5*time.Second,
	); err != nil {
		return err
	}

	g.etcd = k
	// Clear any potential remaining datas from previous gatekeepers
	// WILL prevent multiple gatekeepers to run at the same time.
	_, err := (*k).Delete(context.Background(), "/gatekeeper/slotfs", &client.DeleteOptions{Recursive: true})
	if err != nil && err.(client.Error).Code != client.ErrorCodeKeyNotFound {
		return err
	}
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

// WithHostKey loads the private key at `path` and create a new signer from it.
// if the file does not exist (or can't be read from), a new host key will be
// generated and stored at `path`
func (g *GateKeeper) WithHostKey(path string) error {
	var key *rsa.PrivateKey
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Info().Msg("Generating new host key")
		key, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			log.Error().Str("error", err.Error()).Msg("Failed to generate host key")
			return err
		}
		payload := x509.MarshalPKCS1PrivateKey(key)
		err = ioutil.WriteFile(path, payload, 0600)
		if err != nil {
			log.Warn().
				Str("error", err.Error()).
				Msg("Failed to persist private key, identity WILL change if the gatekeeper is restarted")
		}
	} else {
		log.Debug().Str("path", path).Msg("Imported host key.")
		key, err = x509.ParsePKCS1PrivateKey(b)
		if err != nil {
			log.Error().Str("error", err.Error()).Msg("Failed to parse host key")
			return err
		}
	}
	g.hostKey, err = gossh.NewSignerFromKey(key)
	if err != nil {
		log.Error().Str("error", err.Error()).Msg("Failed to import host key from private key")
		return err
	}
	return nil
}

// initSSHServer creates a new SSH server with
// - routing logic through command
// - reverse port forwarding logic for agents
func (g *GateKeeper) initSSHServer() error {
	if g.srv != nil {
		return errors.New("SSH server already initialized")
	}
	if g.hostKey == nil {
		return errors.New("Host key missing")
	}
	addr := fmt.Sprintf("%s:%d", g.Meta.SSHAddr, g.Meta.SSHPort)
	server := ssh.Server{
		Addr:        addr,
		HostSigners: []ssh.Signer{g.hostKey},
		Handler:     ssh.Handler(g.proxyCommandHandler()),
		ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(g.reversePortForwardHandler(*g.etcd)),
	}
	g.srv = &server
	log.Info().
		Str("addr", g.Meta.SSHAddr).
		Uint16("port", g.Meta.SSHPort).
		Msg("starting SSH server")
	err := server.ListenAndServe()
	if err != nil {
		log.Error().
			Str("error", err.Error()).
			Msg("SSH server exited unexpectedly.")
	}
	return err
}
