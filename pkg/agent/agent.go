package agent

import (
	"crypto/rsa"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path"
	"strings"

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

	privateKey *rsa.PrivateKey
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

func forwardConnection(conn net.Conn, fwd *ForwardedHost) error {
	localHost := fmt.Sprintf("%s:%d", fwd.Host, fwd.Port)
	localConn, err := net.Dial("tcp", localHost)
	if err != nil {
		return err
	}
	go func() {
		log.Info().
			Str("host", localHost).
			Msg("Starting reverse forwarding.")
		go func() {
			io.Copy(conn, localConn)
		}()
		go func() {
			io.Copy(localConn, conn)
		}()
	}()
	return nil
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
	conn, err := ssh.Dial(
		"tcp",
		host+":"+fmt.Sprintf("%d", port),
		sshConfig,
	)
	if err != nil {
		return err
	}
	conn.Start(fwHost.Domain)
	return forwardConnection(conn.Conn, fwHost)
}

// Run is the entrypoint for the agent
func (a *Agent) Run() {
	a.setupFileSystem()
	a.synchronizeIdentities()
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
}
