package ls

import (
	"fmt"
	"os"
	"strings"

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
		Use:   "ls",
		Short: "List identities.",
		Long:  `List agent registered identities in the RSSH root directory.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return parseArgsE(&flags)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.Init(); err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Could not initialize RSSH agent.")
				os.Exit(1)
			}
			fmt.Printf("|%s|\n", strings.Repeat("-", 1+(38*2)))
			fmt.Printf("| %-36s | %-36s | \n", "Domain", "UID")
			fmt.Printf("|%s|\n", strings.Repeat("-", 1+(38*2)))
			a.WalkIdentities(func(fw *agent.ForwardedHost) {
				fmt.Printf("| %-36s | %-36s |\n", fw.Domain, fw.UID)
			})
			fmt.Printf("|%s|\n", strings.Repeat("-", 1+(38*2)))
			return nil
		},
	}

	return cmd
}
