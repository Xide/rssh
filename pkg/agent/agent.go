package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/Xide/rssh/pkg/api"
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
}

// Agent is the main structure of this package, it gets deserialized from
// the configuration file.
type Agent struct {
	Hosts            []ForwardedHost `json:"hosts" mapstructure:"hosts"`
	SecretsDirectory string          `json:"secrets_directory" mapstructure:"secrets_directory"`
}

// RegisterRequest is the
type RegisterRequest struct {
	// Requested domain FQDN (including RSSH root domain)
	Domain string
	// Host to dial for the "local" end of the connection
	Host string
	// Port to dial for the "local" end of the connection
	Port uint16
	// Port on which the API listen to requests on the root domain
	APIPort uint16
}

// registerRequest perform the http request, parse the result,
// interpret any server error and return the generated credentials
// upon success
func registerRequest(url string) (*api.AgentCredentials, error) {
	resp, err := http.Post(
		url,
		"application/json",
		nil,
	)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	registerResponse := api.RegisterResponse{}
	err = json.Unmarshal(body, &registerResponse)

	if err != nil {
		return nil, err
	}

	if registerResponse.Err != nil {
		return nil, errors.New(registerResponse.Err.Msg)
	}
	return registerResponse.AgentID, nil
}

// RegisterHost contact the API to retreive credentials for domain `req.Domain`
func (a *Agent) RegisterHost(req *RegisterRequest) error {
	rootDomain := strings.Join(strings.Split(req.Domain, ".")[1:], ".")
	subDomain := strings.Split(req.Domain, ".")[0]

	log.Debug().
		Str("root", rootDomain).
		Str("sub", subDomain).
		Msg("Registration request")
	creds, err := registerRequest(fmt.Sprintf(
		"http://%s:%d/register/%s",
		rootDomain,
		req.APIPort,
		subDomain,
	))
	if err != nil {
		return err
	}

	err = persistKeyToDisk(".", req.Domain, creds)
	if err != nil {
		return err
	}
	log.Info().
		Str("domain", req.Domain).
		Msg("Persisted credentials to disk.")
	return nil
}

func (a *Agent) secureIdentityDirectory() error {
	if err := os.Chmod(a.SecretsDirectory, 0700); err != nil {
		return err
	}
	return nil
}

// Run is the entrypoint for the agent
func (a *Agent) Run() {
	if err := a.secureIdentityDirectory(); err != nil {
		log.Warn().
			Str("error", err.Error()).
			Msg("Could not secure private keys directory.")
	}

}
