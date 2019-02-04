package gatekeeper

import (
	"errors"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/Xide/rssh/pkg/gatekeeper"
	"github.com/Xide/rssh/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type GatekeeperFlags struct {
	BindAddr      string `mapstructure:"ssh_addr"`
	BindPort      uint16 `mapstructure:"ssh_port"`
	SSHPortRange  string `mapstructure:"ssh_port_range"`
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

func parseArgsE(flags *GatekeeperFlags) error {
	// Shared resource not directly available throught mapstructure
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

func NewCommand(flags *GatekeeperFlags) *cobra.Command {
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
				log.Fatal().
					Str("error", err.Error()).
					Msg("Could not start Gatekeeper")
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
	viper.BindPFlag("api.addr", cmd.Flags().Lookup("addr"))

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

	return cmd
}
