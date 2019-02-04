package agent

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/Xide/rssh/pkg/api"
)

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
