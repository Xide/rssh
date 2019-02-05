package agent

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Xide/rssh/pkg/api"
	"github.com/rs/zerolog/log"
)

func ensureDirectory(path string, mode os.FileMode) error {
	s, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(path, mode)
			if err != nil {
				return err
			}
			log.Debug().Str("directory", path).Msg("Created directory.")
		} else {
			return err
		}
	} else if !s.IsDir() {
		return errors.New("path is not a directory")
	}
	return nil
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

func (a *Agent) ensureRSSHDirectories() error {
	if err := ensureDirectory(a.RootDirectory, 0744); err != nil {
		return err
	}
	if err := ensureDirectory(path.Join(a.RootDirectory, "identities"), 0700); err != nil {
		return err
	}
	return nil
}

func (a *Agent) setupFileSystem() error {
	if err := a.ensureRSSHDirectories(); err != nil {
		log.Warn().
			Str("error", err.Error()).
			Msg("Could not create config directory.")
	}
	return nil
}

func parseFwdHostFromFile(file string) (*ForwardedHost, error) {
	pemEncoded, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(pemEncoded)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("invalid PEM block")
	}
	for _, k := range []string{"uid", "host", "port"} {
		if len(block.Headers[k]) == 0 {
			return nil, fmt.Errorf("invalid %s encoded in private key", k)
		}
	}
	fwPort, err := strconv.ParseUint(block.Headers["port"], 10, 16)
	if err != nil {
		return nil, err
	}

	pkey, err := parsePrivatekeyFromFile(file)
	if err != nil {
		return nil, err
	}

	fwHost := ForwardedHost{
		UID:        block.Headers["uid"],
		Host:       block.Headers["host"],
		Port:       uint16(fwPort),
		Domain:     strings.TrimPrefix(filepath.Base(file), "id_rsa."),
		privateKey: pkey,
	}
	return &fwHost, nil
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
	err := ioutil.WriteFile(path.Join(configDir, keyName), creds.Secret, 0600)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(configDir, keyName+".pub"), creds.Identity, 0644)
	if err != nil {
		return err
	}
	return nil
}
