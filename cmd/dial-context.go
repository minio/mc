package cmd

import (
	"context"
	"net"
	"time"
)

type dialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)

// newDialContext creates a dial context that switches to ipv4 if the ipv4 flag is set.
//
// The parameters are set on the dialer.
func newDialContext(timeout, keepAlive time.Duration, dualStack bool) dialContextFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		d := &net.Dialer{
			Timeout:   timeout * time.Second,
			KeepAlive: keepAlive * time.Second,
			DualStack: dualStack,
		}
		
		// Use v4
		if globalIPv4Only {
			switch network {
			case "tcp", "tcp6":
				network = "tcp4"
			case "udp", "udp6":
				network = "udp4"
			case "ip", "ip6":
				network = "ip4"
			}
		}
		return d.DialContext(ctx, network, address)
	}
}
