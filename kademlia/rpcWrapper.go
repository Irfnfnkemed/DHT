package kademlia

import (
	"errors"

	"github.com/sirupsen/logrus"
)

type RPCWrapper struct {
	node *Node
}

func (wrapper *RPCWrapper) Ping(_Null, _ *Null) error {
	if wrapper.node.Online {
		return nil
	}
	return errors.New("Offline node.")
}

func (wrapper *RPCWrapper) FindNode(pair IpIdPairs, findList *[]string) error {
	list := wrapper.node.FindNode(pair.IdTo)
	*findList = list
	if pair.IpFrom != wrapper.node.IP {
		wrapper.node.flush(pair.IpFrom, true)
	}
	return nil
}

func (wrapper *RPCWrapper) PutIn(pair IpDataPairs, _ *Null) error {
	wrapper.node.PutIn(pair.Datas)
	if pair.IpFrom != wrapper.node.IP {
		wrapper.node.flush(pair.IpFrom, true)
	}
	return nil
}

func (wrapper *RPCWrapper) Getout(pair IpPairs, value *string) error {
	ok := false
	ok, *value = wrapper.node.Getout(pair.IpTo)
	if pair.IpFrom != wrapper.node.IP {
		wrapper.node.flush(pair.IpFrom, true)
	}
	if !ok {
		logrus.Errorf("Getting out error, IP = %s.", wrapper.node.IP)
		return errors.New("Fail to get out data.")
	}
	return nil
}
