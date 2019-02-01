package server

import (
	"strings"

	"github.com/rs/zerolog/log"

	api "github.com/Xide/rssh/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type APIFlags struct {
	BindAddr      string `mapstructure:"addr"`
	BindPort      uint16 `mapstructure:"port"`
	EtcdEndpoints []string
}

// Splits {"a,b", "c"} into {"a", "b", "c"}
// Temporary fix (hopefully) because Cobra doesn't
// handle separators if they are not followed by a whitespace.
func splitParts(maybeParted []string) []string {
	r := []string{}
	for _, x := range maybeParted {
		if strings.Contains(x, ",") {
			for _, newKey := range strings.Split(x, ",") {
				r = append(r, newKey)
			}
		} else {
			r = append(r, x)
		}
	}
	return r
}

// Shared resources not directly available throught mapstructure
func parseArgs(flags *APIFlags) {
	flags.EtcdEndpoints = splitParts(viper.GetStringSlice("etcd.endpoints"))
}

func NewCommand(flags *APIFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the RSSH public server.",
		Long:  `Run the RSSH public server.`,
		PreRun: func(cmd *cobra.Command, args []string) {
			parseArgs(flags)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			httpAPI, err := api.NewDispatcher(
				flags.BindAddr,
				flags.BindPort,
			)
			if err != nil {
				log.Fatal().Str("error", err.Error()).Msg("Failed to start HTTP API dispatcher")
				return err
			}
			executor, err := api.NewExecutor(flags.EtcdEndpoints)
			if err != nil {
				log.Fatal().Str("error", err.Error()).Msg("Failed to start HTTP API executor")
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
