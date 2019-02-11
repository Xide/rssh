package rssh

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Xide/rssh/pkg/utils"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/Xide/rssh/cmd/agent"
	"github.com/Xide/rssh/cmd/api"
	"github.com/Xide/rssh/cmd/gatekeeper"
	"github.com/Xide/rssh/cmd/version"
)

const defaultLevel = zerolog.InfoLevel

type Flags struct {
	LogLevel        string `mapstructure:"log_level"`
	ConfigFile      string
	APIFlags        api.Flags        `mapstructure:"api"`
	GatekeeperFlags gatekeeper.Flags `mapstructure:"gatekeeper"`
	AgentFlags      agent.Flags      `mapstructure:"agent"`
}

func parseLogLevel(strLevel string) zerolog.Level {
	switch strLevel {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		log.Warn().Msg("Invalid log level specified. Ignoring.")
		return defaultLevel
	}
}

func setupLogLevel(flags *Flags) error {
	var fmtLevel func(interface{}) string

	raw := viper.GetString("log_level")
	ll := parseLogLevel(raw)
	isTerminal := terminal.IsTerminal(int(os.Stdout.Fd()))
	if isTerminal {
		fmtLevel = func(i interface{}) string {
			switch i.(string) {
			case "debug":
				return color.New(color.FgHiBlack).SprintFunc()("●")
			case "info":
				return color.New(color.FgBlue).SprintFunc()("●")
			case "warn":
				return color.New(color.FgYellow).SprintFunc()("●")
			case "error":
				return color.New(color.FgRed).SprintFunc()("●")
			case "fatal":
				return color.New(color.FgRed).SprintFunc()("●")
			case "panic":
				return color.New(color.FgRed).SprintFunc()("●")
			default:
				return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
			}
		}
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:         os.Stdout,
		TimeFormat:  time.RFC3339,
		NoColor:     !isTerminal,
		FormatLevel: fmtLevel,
	})
	zerolog.SetGlobalLevel(ll)

	log.Debug().Str("loglevel", raw).Msg("Initialized logging.")
	return nil
}

func NewCommand(flags *Flags) *cobra.Command {
	cobra.OnInitialize(func() {
		utils.InitConfig(flags)
		setupLogLevel(flags)
	})
	cmd := &cobra.Command{
		Use:   "rssh",
		Short: "rssh is a tool for managing reverse shells exposed on a public endpoint.",
		Long:  "rssh is a tool for managing reverse shells exposed on a public endpoint.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		Version: "0.0.1",
	}

	cmd.PersistentFlags().StringVarP(
		&flags.LogLevel,
		"log-level",
		"l",
		defaultLevel.String(),
		"Log level (one of: debug,info,warn,error,fatal,panic)",
	)
	viper.BindPFlag("log_level", cmd.PersistentFlags().Lookup("log-level"))
	cmd.PersistentFlags().StringVar(
		&flags.ConfigFile,
		"config",
		"",
		"path to a custom rssh config file",
	)
	viper.BindPFlag("config", cmd.PersistentFlags().Lookup("config"))

	cmd.AddCommand(version.NewCommand())
	cmd.AddCommand(agent.NewCommand(&flags.AgentFlags))
	cmd.AddCommand(api.NewCommand(&flags.APIFlags))
	cmd.AddCommand(gatekeeper.NewCommand(&flags.GatekeeperFlags))

	return cmd
}

func Execute() {
	configToEnv := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(configToEnv)
	viper.SetEnvPrefix("rssh")
	viper.AutomaticEnv()
	flags := &Flags{}

	if err := NewCommand(flags).Execute(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
