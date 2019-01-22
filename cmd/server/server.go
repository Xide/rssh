package server

import (
	"github.com/spf13/cobra"
	"github.com/Xide/rssh/pkg/server"
)

type apiFlags struct {
	BindAddr string
	BindPort uint16
}

func NewCommand() *cobra.Command {
	flags := &apiFlags{}
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

	return cmd
}