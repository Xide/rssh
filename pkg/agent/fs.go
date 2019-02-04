package agent

import (
	"errors"
	"os"
	"path"

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
		}
	} else if !s.IsDir() {
		return errors.New("path is not a directory")
	}
	return nil
}

func (a *Agent) ensureRSSHDirectories() error {
	if err := ensureDirectory(a.RootDirectory, 0644); err != nil {
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
