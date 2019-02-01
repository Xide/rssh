package gatekeeper

import (
	"errors"
	"fmt"
	"net"

	"github.com/rs/zerolog/log"
)

func findAvailablePort(low uint16, high uint16) (uint16, error) {
	for x := low; x < high; x++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", x))
		if err != nil {
			log.Debug().Uint16("port", x).Msg("findAvailablePort: already in use.")
			continue
		}
		ln.Close()
		log.Debug().Uint16("port", x).Msg("Allocated agent port.")
		return x, nil
	}
	return 0, errors.New("no slot available")
}
