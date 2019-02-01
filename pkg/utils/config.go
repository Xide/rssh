package utils

import (
	"os"
	"os/user"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func InitConfig(flags interface{}) {
	cnf := viper.GetString("config")
	if cnf != "" {
		viper.SetConfigFile(cnf)
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			log.Warn().
				Str("error", err.Error()).
				Msg("Ignoring current directory as config file source.")
		} else {
			viper.AddConfigPath(cwd)
		}

		user, err := user.Current()
		if err != nil {
			log.Warn().
				Str("error", err.Error()).
				Msg("Could not find current user informations, ignoring configuration file")
			return
		}
		viper.AddConfigPath(user.HomeDir)
		viper.SetConfigName(".rssh")
	}
	if err := viper.ReadInConfig(); err == nil {
		if err := viper.Unmarshal(flags); err != nil {
			log.Warn().
				Str("error", err.Error()).
				Msg("Failed to parse configuration")
		}
	} else {
		log.Warn().
			Str("error", err.Error()).
			Msg("Could not load configuration file.")
	}
}
