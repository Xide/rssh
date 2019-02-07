package gatekeeper

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog/log"
	"go.etcd.io/etcd/client"
)

func (g *GateKeeper) reversePortForwardHandler(etcd client.KeysAPI) func(ssh.Context, string, uint32) bool {
	return func(ctx ssh.Context, host string, port uint32) bool {
		log.Debug().
			Str("client_addr", host).
			Uint32("port", port).
			Msg("Port forward request")
		slotFs, err := etcd.Get(context.Background(), "/gatekeeper/slotfs", nil)
		if err != nil {
			log.Error().
				Str("error", err.Error()).
				Str("client_addr", host).
				Uint32("port", port).
				Msg("Failed to load Gk slotFS")
			return false
		}
		if slotFs == nil || slotFs.Node == nil {
			log.Error().
				Str("error", err.Error()).
				Str("client_addr", host).
				Uint32("port", port).
				Msg("Request while Gk slotFS does not exists.")
			return false
		}
		for _, rSlot := range slotFs.Node.Nodes {
			slot := &AgentSlot{}
			sl := strings.Split(rSlot.Key, "/")
			usedPort, err := strconv.ParseUint(sl[len(sl)-1], 10, 16)
			if err != nil {
				log.Warn().
					Str("error", err.Error()).
					Str("client_addr", host).
					Uint32("port", port).
					Msg("Unable to deserialize slot port from etcd.")
				continue
			}
			err = json.Unmarshal([]byte(rSlot.Value), slot)
			if err != nil {
				log.Warn().
					Str("error", err.Error()).
					Str("client_addr", host).
					Uint32("port", port).
					Msg("Unable to deserialize slot from etcd.")
				continue
			}
			if uint16(port) == uint16(usedPort) {
				if slot.Established {
					log.Debug().
						Str("client_addr", host).
						Uint32("port", port).
						Msg("A session is already established for this agent.")
					return false
				}
				slot.Established = true
				payload, err := json.Marshal(slot)
				if err != nil {
					log.Warn().
						Str("client_addr", host).
						Str("error", err.Error()).
						Uint32("port", port).
						Msg("Failed to reserve marshal slot for etcd.")

				}
				if _, err = etcd.Set(context.Background(), rSlot.Key, string(payload), nil); err != nil {
					log.Warn().
						Str("client_addr", host).
						Str("error", err.Error()).
						Uint32("port", port).
						Msg("Failed to reserve establish slot in etcd.")
					return false
				}
				go g.collectClosedSession(ctx, slot)
				log.Debug().
					Str("client_addr", host).
					Uint32("port", port).
					Msg("Accepted port forward")
				return true
			}
		}

		log.Debug().
			Str("client_addr", host).
			Uint32("port", port).
			Str("error", "slot not found").
			Msg("Denied port forward.")
		return false
	}
}

// func (g *GateKeeper) monitorSlots() error {
// 	watcherOpts := client.WatcherOptions{AfterIndex: 0, Recursive: true}
// 	w := (*g.etcd).Watcher("/gatekeeper/slotfs", &watcherOpts)
// 	x, err := w.Next(context.Background())
// 	if err != nil {
// 		return err
// 	}
// 	for {
// 		fmt.Println(x)
// 		x, err = w.Next(context.Background())
// 		if err != nil {
// 			return err
// 		}
// 	}
// }
