package kademlia

import (
	"errors"
)

type RPCWrapper struct {
	node *Node
}

func (wrapper *RPCWrapper) Ping(ipFrom string, _ *Null) error {
	if wrapper.node.Online {
		if ipFrom != wrapper.node.IP {
			wrapper.node.flush(ipFrom)
		}
		return nil
	}
	return errors.New("Offline node.")
}
