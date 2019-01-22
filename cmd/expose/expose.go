package expose

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "expose",
		Short: "Expose your SSH server.",
		Long:  `Expose your SSH server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg("Exposing server to the world")
			return nil
		},
	}
	return cmd
}
