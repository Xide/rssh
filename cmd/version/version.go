package version

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// Version is super useless at the moment
const Version = "0.0.1"

// NewCommand Run() will log the version above and exit.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "prints the CLI version",
		Long:  "prints the CLI version",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().Msg(Version)
			return nil
		},
	}
	return cmd
}
