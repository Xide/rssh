package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/Xide/rssh/pkg/utils"

	"github.com/Xide/rssh/pkg/api"
	"github.com/rs/zerolog/log"
)

// RegisterRequest is the result of a `rssh agent register ...` command.
type RegisterRequest struct {
	// Requested domain FQDN (including RSSH root domain)
	Domain string
	// Host to dial for the "local" end of the connection
	Host string
	// Port to dial for the "local" end of the connection
	Port uint16
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
	subDomain, rootDomain := utils.SplitDomainRequest(req.Domain)

	log.Debug().
		Str("root", rootDomain).
		Str("sub", subDomain).
		Msg("Registration request")
	creds, err := registerRequest(fmt.Sprintf(
		"http://%s:%d/register/%s",
		rootDomain,
		a.APIPort,
		subDomain,
	))
	if err != nil {
		return err
	}

	err = embedHostConfiguration(creds, req)
	if err != nil {
		return err
	}

	err = persistKeyToDisk(path.Join(a.RootDirectory, "identities"), req.Domain, creds)
	if err != nil {
		return err
	}
	log.Info().
		Str("domain", req.Domain).
		Msg("Persisted credentials to disk.")
	return a.synchronizeIdentities()
}
