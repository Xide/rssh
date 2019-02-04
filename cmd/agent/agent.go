package agent

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/Xide/rssh/cmd/agent/register"
	"github.com/Xide/rssh/pkg/agent"
)

type AgentFlags = agent.Agent

func NewCommand(flags *AgentFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Expose your SSH server.",
		Long:  `Expose your SSH server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Run()
			log.Info().Msg("Exposing server to the world")
			return nil
		},
	}

	cmd.AddCommand(register.NewCommand())
	return cmd
}
