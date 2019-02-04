package register

import (
	"errors"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Flags are the command line / environment flags
// accepted by the `rssh agent register` command.
type Flags struct {
	// Requested domain FQDN (including RSSH root domain)
	Domain string
	// Host to expose
	Host string
	// Port to expose
	Port uint16
}

func parseArgsE(flags *Flags) error {
	flags.Domain = viper.GetString("register.domain")
	if len(flags.Domain) == 0 {
		return errors.New("domain is mandatory")
	}

	flags.Host = viper.GetString("register.host")
	p, err := strconv.ParseUint(viper.GetString("register.port"), 10, 16)
	if err != nil {
		return err
	}
	flags.Port = uint16(p)
	return nil
}

// NewCommand return the agent registration cobra command
func NewCommand() *cobra.Command {
	flags := Flags{}
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new endpoint to expose.",
		Long:  `Register a new endpoint to expose.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return parseArgsE(&flags)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().
				Str("domain", flags.Domain).
				Str("Host", flags.Host).
				Uint16("Port", flags.Port).
				Msg("Register new endpoint")
			return nil
		},
	}

	cmd.Flags().StringVarP(
		&flags.Domain,
		"domain",
		"d",
		"",
		"Domain to register",
	)
	viper.BindPFlag("register.domain", cmd.Flags().Lookup("domain"))

	cmd.Flags().StringVarP(
		&flags.Host,
		"host",
		"a",
		"127.0.0.1",
		"Host to expose throught agent",
	)
	viper.BindPFlag("register.host", cmd.Flags().Lookup("host"))

	cmd.Flags().Uint16VarP(
		&flags.Port,
		"port",
		"p",
		22,
		"Port to expose throught agent",
	)
	viper.BindPFlag("register.port", cmd.Flags().Lookup("port"))

	return cmd
}
