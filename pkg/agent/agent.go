package agent

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
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
	// UUID assigned to the agent
	UID string

	privateKey *rsa.PrivateKey
}

// Agent is the main structure of this package, it gets deserialized from
// the configuration file.
type Agent struct {
	Hosts         []ForwardedHost `json:"hosts" mapstructure:"hosts"`
	RootDirectory string          `json:"root_directory" mapstructure:"root_directory"`
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

	err = persistKeyToDisk(path.Join(a.RootDirectory, "identities"), req.Domain, creds)
	if err != nil {
		return err
	}
	log.Info().
		Str("domain", req.Domain).
		Msg("Persisted credentials to disk.")
	return nil
}

func parseUIDFromFile(file string) (string, error) {
	pemEncoded, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(pemEncoded)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return "", errors.New("invalid PEM block")
	}
	if len(block.Headers["uid"]) == 0 {
		return "", errors.New("invalid UID encoded in private key")
	}
	return block.Headers["uid"], nil
}

func parsePrivatekeyFromFile(file string) (*rsa.PrivateKey, error) {
	pemEncoded, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemEncoded)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("invalid PEM block")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

func (a *Agent) orphanKeys() ([]string, error) {
	orphans := []string{}
	keys, err := filterPublicKeys(path.Join(a.RootDirectory, "identities"))
	if err != nil {
		return nil, err
	}
	for _, x := range keys {
		found := false
		for _, y := range a.Hosts {
			if strings.Index(x, y.Domain) != -1 {
				found = true
				break
			}
		}
		if !found {
			orphans = append(orphans, x)
		}
	}
	return orphans, nil
}

func (a *Agent) synchronizeIdentities() error {
	for i, host := range a.Hosts {
		idFile := fmt.Sprintf("id_rsa.%s", host.Domain)
		id, err := parseUIDFromFile(
			path.Join(
				a.RootDirectory,
				"identitites",
				idFile,
			),
		)
		if err != nil {
			log.Warn().
				Str("error", err.Error()).
				Str("file", idFile).
				Msg("Could not load identity")
			a.Hosts = append(a.Hosts[:i], a.Hosts[i+1:]...)
			continue
		}
		a.Hosts[i].UID = id

		key, err := parsePrivatekeyFromFile(
			path.Join(
				a.RootDirectory,
				"identitites",
				idFile,
			),
		)
		if err != nil {
			log.Warn().
				Str("error", err.Error()).
				Str("file", idFile).
				Msg("Could not parse identity secret key")
			a.Hosts = append(a.Hosts[:i], a.Hosts[i+1:]...)
			continue
		}
		a.Hosts[i].privateKey = key
	}
	return nil
}

func filterPublicKeys(path string) ([]string, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	res := []string{}
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".pub") {
			res = append(res, f.Name())
		}
	}
	return res, nil
}

func (a *Agent) loadIdentities() error {
	err := a.synchronizeIdentities()
	if err != nil {
		return err
	}
	orphans, err := a.orphanKeys()
	if err != nil {
		return err
	}
	l := log.Debug()
	for i, x := range orphans {
		l.Str(fmt.Sprintf("orphan-%d", i), x)
	}
	l.Msg("Orphan keys (identities witout configuration) found.")
	return nil
}

// Run is the entrypoint for the agent
func (a *Agent) Run() {
	a.setupFileSystem()
	a.loadIdentities()
}
