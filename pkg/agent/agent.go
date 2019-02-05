package agent

import (
	"crypto/rsa"

	"github.com/rs/zerolog/log"
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

// Run is the entrypoint for the agent
func (a *Agent) Run() {
	a.setupFileSystem()
	a.synchronizeIdentities()
	log.Info().
		Int("hosts_count", len(a.hosts)).
		Msg("Finished hosts import.")
}
