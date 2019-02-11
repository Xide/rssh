package rm

import (
	"os"

	"github.com/Xide/rssh/pkg/agent"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type Flags = agent.Agent

func parseArgsE(flags *Flags) error {
	return nil
}

// NewCommand return the identity list cobra command
func NewCommand(a *agent.Agent) *cobra.Command {
	flags := Flags{}
	cmd := &cobra.Command{
		Use:   "rm",
		Short: "Remove identities.",
		Long:  `Remove identities (by domain or UID).`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return parseArgsE(&flags)
		},
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.Init(); err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Could not initialize RSSH agent.")
				os.Exit(1)
			}
			for _, x := range args {
				if err := a.RemoveIdentity(x); err != nil {
					log.Warn().Str("error", err.Error()).Msg("Could not remove identity")
				} else {
					log.Info().Msg("Identity removed")
				}
			}
			return nil
		},
	}

	return cmd
}
