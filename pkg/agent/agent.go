package agent

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

// ForwardedHost describe a socket exposed by the agent
// through the gatekeeper.
type ForwardedHost struct {
	// complete FQDN for which the host is bound
	Domain string `json:"domain" mapstructure:"domain"`
	// Address or domain on which the agent will dial the connection
	Host string `json:"host" mapstructure:"host"`
	// Port on which the agent will connect to
	Port uint16 `json:"port" mapstructure:"ports"`
	// UUID assigned to the agent
	UID string

	privateKey     *rsa.PrivateKey
	gatekeeperPort uint16
}

// Agent is the main structure of this package, it gets deserialized from
// the configuration file.
type Agent struct {
	hosts         []ForwardedHost
	RootDirectory string `json:"root_directory" mapstructure:"root_directory"`
}

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func forwardConnection(conn ssh.Channel, fwd *ForwardedHost) error {
	localHost := fmt.Sprintf("%s:%d", fwd.Host, fwd.Port)
	localConn, err := net.Dial("tcp", localHost)
	if err != nil {
		return err
	}
	go func() {
		defer conn.Close()
		defer localConn.Close()
		io.Copy(conn, localConn)
	}()
	go func() {
		defer conn.Close()
		defer localConn.Close()
		io.Copy(localConn, conn)
	}()
	return nil
}

func handleNewConnections(ch <-chan ssh.NewChannel, fwHost *ForwardedHost) {
	for {
		select {
		case x := <-ch:
			log.Debug().
				Str("domain", fwHost.Domain).
				Msg("New connection request.")
			ch, _, err := x.Accept()
			if err != nil {
				log.Warn().
					Str("error", err.Error()).
					Str("domain", fwHost.Domain).
					Msg("Failed to accept new connection.")
			}
			err = forwardConnection(ch, fwHost)
			if err != nil {
				log.Warn().
					Str("error", err.Error()).
					Str("domain", fwHost.Domain).
					Msg("Failed to forward new connection.")
			}

		}
	}
}

func (a *Agent) establishReverseForward(host string, port uint16, fwHost *ForwardedHost) error {
	log.Debug().
		Str("domain", fwHost.Domain).
		Msg("Setup socket for agent.")
	pubKeyFile := path.Join(
		a.RootDirectory,
		"identities",
		fmt.Sprintf("id_rsa.%s.pub", fwHost.Domain),
	)

	sshConfig := &ssh.ClientConfig{
		User: "rssh_agent",
		Auth: []ssh.AuthMethod{
			publicKeyFile(pubKeyFile),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	gkAddr := fmt.Sprintf("%s:%d", fwHost.Host, fwHost.Port)
	conn, err := net.Dial("tcp", gkAddr)
	if err != nil {
		return err
	}
	sshConn, ch, _, err := ssh.NewClientConn(conn, gkAddr, sshConfig)
	if err != nil {
		return err
	}

	c, data, err := sshConn.SendRequest("tcpip-forward", true, ssh.Marshal(&struct {
		BindAddr string
		BindPort uint32
	}{
		// TODO: Change hardcoded values
		BindAddr: "127.0.0.1",
		BindPort: 12346,
	}))
	if err != nil {
		return err
	}
	if c {
		// TODO: Change hardcoded values
		cn, err := net.Dial("tcp", "127.0.0.1:12346")
		if err != nil {
			sshConn.Close()
			return err
		}
		cn.Close()
		go handleNewConnections(ch, fwHost)
	} else {
		log.Error().
			Str("response", string(data)).
			Msg("Failed to request port forwarding.")
		return errors.New(string(data))
	}
	return nil
}

// Init stup the identities and directories required by the agent.
func (a *Agent) Init() error {
	if err := a.setupFileSystem(); err != nil {
		return err
	}
	if err := a.synchronizeIdentities(); err != nil {
		return err
	}
	return nil
}

// Run is the entrypoint for the agent
func (a *Agent) Run() {
	a.Init()
	log.Info().
		Int("hosts_count", len(a.hosts)).
		Msg("Finished hosts import.")
	for _, credential := range a.hosts {
		root := strings.Join(strings.Split(credential.Domain, ".")[1:], ".")
		err := a.establishReverseForward(root, 2223, &credential)
		if err != nil {
			log.Warn().
				Str("error", err.Error()).
				Str("uid", credential.UID).
				Msg("Failed to establish reverse forward")
		}
	}
	for {
		time.Sleep(3600 * time.Second)
	}
}
