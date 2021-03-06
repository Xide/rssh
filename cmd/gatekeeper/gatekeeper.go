package gatekeeper

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/Xide/rssh/pkg/gatekeeper"
	"github.com/Xide/rssh/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Flags are injected by parent command
// from the cli > env > config file > defaults
type Flags struct {
	BindAddr      string `mapstructure:"ssh_addr"`
	BindPort      uint16 `mapstructure:"ssh_port"`
	SSHPortRange  string `mapstructure:"ssh_port_range"`
	HostKeyFile   string `mapstructure:"ssh_host_key"`
	SSHPortLow    uint16
	SSHPortHigh   uint16
	EtcdEndpoints []string
}

func parsePortRange(raw string) (uint16, uint16, error) {
	ports := strings.Split(raw, "-")
	if len(ports) != 2 {
		return 0, 0, errors.New("Invalid port range format : expected two dash separated integers")
	}
	low, err := strconv.ParseUint(ports[0], 10, 16)
	if err != nil {
		return 0, 0, errors.New("first port is not a base 10 integer")
	}

	high, err := strconv.ParseUint(ports[1], 10, 16)
	if err != nil {
		return 0, 0, errors.New("second port is not a base 10 integer")
	}

	return utils.Min(uint16(low), uint16(high)), utils.Max(uint16(low), uint16(high)), nil
}

func parseArgsE(flags *Flags) error {
	// Shared resource not directly available through mapstructure
	flags.EtcdEndpoints = utils.SplitParts(viper.GetStringSlice("etcd.endpoints"))

	// SSH port range parsing
	pRangeLow, pRangeHigh, err := parsePortRange(viper.GetString("gatekeeper.ssh_port_range"))
	if err != nil {
		log.Error().
			Str("error", err.Error()).
			Str("port-range", viper.GetString("gatekeeper.ssh_port_range")).
			Msg("Could not parse SSH port range.")
		return err
	}
	flags.SSHPortLow = pRangeLow
	flags.SSHPortHigh = pRangeHigh
	return nil
}

// NewCommand is the Gatekeeper CLI entrypoint
// it will block upon returned command Run()
func NewCommand(flags *Flags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gatekeeper",
		Short: "Run the RSSH public ssh proxy.",
		Long:  `Run the RSSH public ssh proxy.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return parseArgsE(flags)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info().
				Str("addr", flags.BindAddr).
				Uint16("port", flags.BindPort).
				Str("port-range", flags.SSHPortRange).
				Msg("Starting Gatekeeper")
			g, err := gatekeeper.NewGateKeeper(flags.BindAddr, flags.BindPort)
			if err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Could not start Gatekeeper")
			}

			if err := g.WithEtcdE(flags.EtcdEndpoints); err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Etcd unreachable")
				os.Exit(1)
			}

			if err := g.WithHostKey(flags.HostKeyFile); err != nil {
				log.Error().
					Str("error", err.Error()).
					Msg("Failed to generate host key")
				os.Exit(1)
			}

			return g.Run()
		},
	}

	cmd.Flags().StringVarP(
		&flags.BindAddr,
		"addr",
		"a",
		"0.0.0.0",
		"SSH server address",
	)
	viper.BindPFlag("gatekeeper.ssh_addr", cmd.Flags().Lookup("addr"))

	cmd.Flags().Uint16VarP(
		&flags.BindPort,
		"port",
		"p",
		2223,
		"SSH server port",
	)
	viper.BindPFlag("gatekeeper.port", cmd.Flags().Lookup("port"))

	cmd.Flags().StringSliceVarP(
		&flags.EtcdEndpoints,
		"etcd",
		"e",
		[]string{"http://127.0.0.1:2379"},
		"Comma separated list of the Etcd hosts to discover",
	)
	viper.BindPFlag("etcd.endpoints", cmd.Flags().Lookup("etcd"))

	cmd.Flags().StringVarP(
		&flags.SSHPortRange,
		"port-range",
		"r",
		"31240-65535",
		"Port range where RSSH will bind the agents listener on (format: '$min-$max')",
	)
	viper.BindPFlag("gatekeeper.ssh_port_range", cmd.Flags().Lookup("port-range"))

	cmd.Flags().StringVarP(
		&flags.HostKeyFile,
		"host-key",
		"i",
		".rssh-gk-host.key",
		"SSH server host file. If the destination file does not exists, a new one will be generated there.",
	)
	viper.BindPFlag("gatekeeper.ssh_host_key", cmd.Flags().Lookup("host-key"))

	return cmd
}
