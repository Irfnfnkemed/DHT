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

func (wrapper *RPCWrapper) FindNode(ips callArgs, findList *[]string) error {
	ipTo := ips.args.(string)
	list := wrapper.node.FindNode(ipTo)
	for _, ipFind := range list {
		*findList = append(*findList, ipFind)
	}
	if ips.ipFrom != wrapper.node.IP {
		wrapper.node.flush(ips.ipFrom)
	}
	return nil
}
