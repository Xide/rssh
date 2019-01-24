package server

import (
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"

	api "github.com/Xide/rssh/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type apiFlags struct {
	BindAddr      string
	BindPort      uint16
	EtcdEndpoints []string
}

func parseArgs(flags *apiFlags) func() {
	return func() {
		flags.BindAddr = viper.Get("addr").(string)
		port, err := strconv.ParseUint(viper.Get("port").(string), 10, 16)
		if err != nil {
			log.Fatal().
				Str("port", viper.Get("addr").(string)).
				Msg(fmt.Sprintf("Could not parse %s as an integer.", viper.Get("addr").(string)))
		}
		flags.BindPort = uint16(port)
	}
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal().Str("error", err.Error()).Msg("Could not find user home directory")
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".")
	}
	viper.AutomaticEnv() // read in environment variables that match
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func NewCommand() *cobra.Command {
	flags := &apiFlags{}

	cobra.OnInitialize(parseArgs(flags))
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the RSSH public server.",
		Long:  `Run the RSSH public server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			httpAPI, err := api.NewDispatcher(
				flags.BindAddr,
				flags.BindPort,
			)
			if err != nil {
				return err
			}

			executor, err := api.NewExecutor([]string{"http://127.0.0.1:2379"})
			if err != nil {
				return err
			}
			return httpAPI.Run(executor)
		},
	}

	cmd.PersistentFlags().StringVarP(
		&flags.BindAddr,
		"addr",
		"a",
		"0.0.0.0",
		"HTTP API bind address",
	)

	cmd.PersistentFlags().Uint16VarP(
		&flags.BindPort,
		"port",
		"p",
		8080,
		"HTTP API port",
	)

	viper.BindPFlags(cmd.PersistentFlags())
	return cmd
}
