package gatekeeper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gliderlabs/ssh"
	"github.com/rs/zerolog/log"
	"go.etcd.io/etcd/client"
)

func (g *GateKeeper) getSlotFS() (*client.Nodes, error) {
	slotFs, err := (*g.etcd).Get(context.Background(), "/gatekeeper/slotfs", nil)
	if err != nil {
		log.Error().
			Str("error", err.Error()).
			Msg("Failed to load Gk slotFS")
		return nil, err
	}
	if slotFs == nil || slotFs.Node == nil {
		log.Error().
			Str("error", err.Error()).
			Msg("Request while Gk slotFS does not exists.")
		return nil, errors.New("empty gatekeeper slotFS")
	}
	return &slotFs.Node.Nodes, nil
}

func (g *GateKeeper) getSlot(etcdNode *client.Node) (*AgentSlot, error) {
	slot := &AgentSlot{}
	err := json.Unmarshal([]byte(etcdNode.Value), slot)
	if err != nil {
		return nil, err
	}
	return slot, nil
}

func (g *GateKeeper) setSlot(slot *AgentSlot, key string) error {
	payload, err := json.Marshal(slot)
	if err != nil {
		return err
	}
	if _, err = (*g.etcd).Set(context.Background(), key, string(payload), nil); err != nil {
		return err
	}
	return nil
}

func (g *GateKeeper) getFirstSlotForFn(fn func(*AgentSlot) bool) (*AgentSlot, error) {
	slots, err := g.getSlotFS()
	if err != nil {
		return nil, err
	}
	for _, rSlot := range *slots {
		slot, err := g.getSlot(rSlot)
		if err != nil {
			log.Warn().
				Str("error", err.Error()).
				Msg("Unable to deserialize slot from etcd.")
			continue
		}
		if fn(slot) {
			return slot, nil
		}
	}
	return nil, fmt.Errorf("getFirstSlotForFn: nothing matched in slotFS")
}

func (g *GateKeeper) getSlotForPort(port uint16) (*AgentSlot, error) {
	return g.getFirstSlotForFn(func(sl *AgentSlot) bool {
		return uint16(port) == sl.Port
	})
}

func (g *GateKeeper) setSlotForPort(slot *AgentSlot, port uint16) error {
	return g.setSlot(slot, fmt.Sprintf("/gatekeeper/slotfs/%d", port))
}

func (g *GateKeeper) reversePortForwardHandler(etcd client.KeysAPI) func(ssh.Context, string, uint32) bool {
	return func(ctx ssh.Context, host string, port uint32) bool {
		log.Debug().
			Str("client_addr", host).
			Uint32("port", port).
			Msg("Port forward request")

		slot, err := g.getSlotForPort(uint16(port))
		if err != nil {
			log.Debug().
				Str("client_addr", host).
				Uint32("port", port).
				Str("error", "slot not found").
				Msg("Denied port forward.")
			return false
		}

		if slot.Established {
			log.Debug().
				Str("client_addr", host).
				Uint32("port", port).
				Msg("A session is already established for this agent.")
			return false
		}
		slot.Established = true

		if err = g.setSlotForPort(slot, uint16(port)); err != nil {
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
