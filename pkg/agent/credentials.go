package agent

import (
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/Xide/rssh/pkg/api"
	"github.com/rs/zerolog/log"
)

func embedHostConfiguration(creds *api.AgentCredentials, req *RegisterRequest) error {
	block, _ := pem.Decode(creds.Secret)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return errors.New("invalid PEM block")
	}
	block.Headers["host"] = req.Host
	block.Headers["port"] = strconv.FormatUint(uint64(req.Port), 10)
	creds.Secret = pem.EncodeToMemory(block)
	return nil
}

func (a *Agent) synchronizeIdentities() error {
	hosts := []ForwardedHost{}
	keys, err := filterPublicKeys(path.Join(a.RootDirectory, "identities"))
	if err != nil {
		return err
	}
	for _, idFile := range keys {
		fw, err := parseFwdHostFromFile(
			path.Join(
				a.RootDirectory,
				"identities",
				idFile,
			),
		)
		if err != nil {
			log.Warn().
				Str("error", err.Error()).
				Str("file", idFile).
				Msg("Could not load identity")
			continue
		}
		if a.isImported(fw) {
			continue
		}
		hosts = append(hosts, *fw)
		log.Debug().
			Str("identity", fw.UID).
			Str("file", idFile).
			Msg("Identity imported.")
	}
	a.hosts = append(a.hosts, hosts...)
	return nil
}

func (a *Agent) isImported(fwHost *ForwardedHost) bool {
	for _, x := range a.hosts {
		if fwHost.UID == x.UID {
			return true
		}
	}
	return false
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

func (a *Agent) RemoveIdentity(uid string) error {
	for _, x := range a.hosts {
		if uid == x.UID || uid == x.Domain {
			path := path.Join(
				a.RootDirectory,
				"identities",
				"id_rsa."+x.Domain,
			)
			err := os.RemoveAll(path)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return errors.New("Identity not found : " + uid)
}
