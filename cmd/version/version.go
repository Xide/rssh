package version

import (
	"github.com/spf13/cobra"
	"github.com/rs/zerolog/log"

)

const Version = "0.0.1"

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