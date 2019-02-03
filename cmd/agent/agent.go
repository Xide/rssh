package expose

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type exposeFlags struct {
	Config string
	Domain string
}

func NewCommand() *cobra.Command {
	flags := exposeFlags{}
	cmd := &cobra.Command{
		Use:   "expose",
		Short: "Expose your SSH server.",
		Long:  `Expose your SSH server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg("Exposing server to the world")
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(
		&flags.Domain,
		"domain",
		"d",
		"",
		"Subdomain on which the agent will be exposed.",
	)

	cmd.PersistentFlags().StringVarP(
		&flags.Config,
		"config",
		"c",
		"",
		"Server configuration file to use",
	)
	viper.BindPFlags(cmd.PersistentFlags())
	return cmd
}
