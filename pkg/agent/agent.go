package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Xide/rssh/pkg/api"
	"github.com/rs/zerolog/log"
)

type ForwardedHost struct {
	Domain string `json:"domain" mapstructure:"domain"`
	Host   string `json:"host" mapstructure:"host"`
	Port   uint16 `json:"port" mapstructure:"ports"`
}

type Agent struct {
	Hosts            []ForwardedHost `json:"hosts" mapstructure:"hosts"`
	SecretsDirectory string          `json:"secrets_directory" mapstructure:"secrets_directory"`
}

type RegisterRequest struct {
	// Requested domain FQDN (including RSSH root domain)
	Domain string
	// Host to expose
	Host string
	// Port to expose
	Port uint16
	// Port on which the API listen to requests on the root domain
	APIPort uint16
}

func persistKeyToDisk(
	configDir string,
	domain string,
	creds *api.AgentCredentials,
) error {
	if creds == nil {
		return errors.New("empty credentials cannot be persisted")
	}
	keyName := fmt.Sprintf("id_rsa.%s", domain)
	err := ioutil.WriteFile(keyName, creds.Secret, 0600)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(keyName+".pub", creds.Identity, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (a *Agent) RegisterHost(req *RegisterRequest) error {
	rootDomain := strings.Join(strings.Split(req.Domain, ".")[1:], ".")
	subDomain := strings.Split(req.Domain, ".")[0]

	log.Debug().
		Str("root", rootDomain).
		Str("sub", subDomain).
		Msg("Registration request")
	resp, err := http.Post(
		fmt.Sprintf(
			"http://%s:%d/register/%s",
			rootDomain,
			req.APIPort,
			subDomain,
		),
		"application/json",
		nil,
	)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	registerResponse := api.RegisterResponse{}
	err = json.Unmarshal(body, &registerResponse)

	if err != nil {
		return err
	}

	if registerResponse.Err != nil {
		return errors.New(registerResponse.Err.Msg)
	}

	err = persistKeyToDisk(".", req.Domain, registerResponse.AgentID)
	if err != nil {
		return err
	}
	log.Info().
		Str("domain", req.Domain).
		Msg("Persisted credentials to disk.")
	return nil
}

func (a *Agent) Run() {

}
