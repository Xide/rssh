package agent

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Xide/rssh/cmd/agent/register"
	"github.com/Xide/rssh/pkg/agent"
)

// AgentFlags unmarshall directly to the agent definition
type AgentFlags = agent.Agent

// NewCommand return the agent entrypoint command
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

	cmd.PersistentFlags().StringVarP(
		&flags.SecretsDirectory,
		"secrets-dir",
		"s",
		"",
		"Directory used to store secret keys",
	)
	viper.BindPFlag("agent.secrets_directory", cmd.PersistentFlags().Lookup("secrets-dir"))

	cmd.AddCommand(register.NewCommand(flags))
	return cmd
}
