package rssh

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"time"

	"github.com/Xide/rssh/cmd/expose"
	"github.com/Xide/rssh/cmd/server"
	"github.com/Xide/rssh/cmd/version"
)

const defaultLevel = zerolog.InfoLevel

type Flags struct {
	LogLevel   string
	ConfigFile string
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

func setupLogLevel(flags *Flags, cmd *cobra.Command, args []string) error {
	ll := parseLogLevel(flags.LogLevel)
	zerolog.SetGlobalLevel(ll)
	log.Debug().Str("loglevel", flags.LogLevel).Msg("Initialized logging.")
	return nil
}

func NewCommand() *cobra.Command {
	flags := &Flags{}
	cmd := &cobra.Command{
		Use:   "rssh",
		Short: "rssh is a tool for managing reverse shells exposed on a public endpoint.",
		Long:  "rssh is a tool for managing reverse shells exposed on a public endpoint.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return setupLogLevel(flags, cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		Version: "0.0.1",
	}

	cmd.PersistentFlags().StringVar(
		&flags.LogLevel,
		"loglevel",
		defaultLevel.String(),
		"Log level (one of: debug,info,warn,error,fatal,panic)",
	)

	cmd.AddCommand(version.NewCommand())
	cmd.AddCommand(server.NewCommand())
	cmd.AddCommand(expose.NewCommand())

	return cmd
}

func Execute() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
		NoColor:    !terminal.IsTerminal(int(os.Stdout.Fd())),
	})

	if err := NewCommand().Execute(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
