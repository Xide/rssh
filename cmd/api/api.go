package api

import (
	"errors"
	"os"

	"github.com/rs/zerolog/log"

	"github.com/Xide/rssh/pkg/api"
	"github.com/Xide/rssh/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Flags struct {
	BindAddr      string `mapstructure:"addr"`
	BindPort      uint16 `mapstructure:"port"`
	RootDomain    string `mapstructure:"domain"`
	EtcdEndpoints []string
}

func parseArgs(flags *Flags) error {
	// Shared resource not directly available through mapstructure
	flags.EtcdEndpoints = utils.SplitParts(viper.GetStringSlice("etcd.endpoints"))

	// Domain validation
	if !utils.IsValidDomain(flags.RootDomain) {
		log.Error().
			Str("domain", flags.RootDomain).
			Msg("Invalid domain name.")
		return errors.New("invalid domain")
	}
	return nil
}

func NewCommand(flags *Flags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Run the RSSH public HTTP API.",
		Long:  `Run the RSSH public HTTP API.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return parseArgs(flags)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			httpAPI, err := api.NewDispatcher(
				flags.BindAddr,
				flags.BindPort,
				flags.RootDomain,
				flags.EtcdEndpoints,
			)
			if err != nil {
				log.Error().Str("error", err.Error()).Msg("Failed to start HTTP API dispatcher")
				os.Exit(1)
			}
			err = httpAPI.Run()
			if err != nil {
				log.Error().Str("error", err.Error()).Msg("API server failed unexpectedly")
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(
		&flags.BindAddr,
		"addr",
		"a",
		"0.0.0.0",
		"HTTP API bind address",
	)
	viper.BindPFlag("api.addr", cmd.PersistentFlags().Lookup("addr"))

	cmd.PersistentFlags().Uint16VarP(
		&flags.BindPort,
		"port",
		"p",
		9321,
		"HTTP API port",
	)
	viper.BindPFlag("api.port", cmd.PersistentFlags().Lookup("port"))

	cmd.PersistentFlags().StringSliceVarP(
		&flags.EtcdEndpoints,
		"etcd",
		"e",
		[]string{"http://127.0.0.1:2379"},
		"Comma separated list of the Etcd hosts to discover",
	)
	viper.BindPFlag("etcd.endpoints", cmd.PersistentFlags().Lookup("etcd"))

	return cmd
}
