package agent

import (
	"os"
	"os/user"
	"path"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Xide/rssh/cmd/agent/register"
	"github.com/Xide/rssh/pkg/agent"
)

// Flags unmarshall directly to the agent definition
type Flags = agent.Agent

func getRSSHBaseDirectory() string {
	user, err := user.Current()
	if err != nil {
		cwd, err := os.Getwd()
		if err != nil {
			return "/etc/rssh"
		}
		return cwd
	}
	return path.Join(user.HomeDir, ".rssh")
}

// NewCommand return the agent entrypoint command
func NewCommand(flags *Flags) *cobra.Command {
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
		path.Join(getRSSHBaseDirectory(), "identities"),
		"Directory used to store secret keys",
	)
	viper.BindPFlag("agent.secrets_directory", cmd.PersistentFlags().Lookup("secrets-dir"))

	cmd.AddCommand(register.NewCommand(flags))
	return cmd
}
