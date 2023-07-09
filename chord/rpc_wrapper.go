package chord

import (
	"errors"
	"math/big"
)

type RPC_wrapper struct {
	node *Node
}

func (wrapper *RPC_wrapper) Find_successor(id *big.Int, ip *string) error {
	return wrapper.node.Find_successor(id, ip)
}

func (wrapper *RPC_wrapper) Find_predecessor(id *big.Int, ip *string) error {
	return wrapper.node.Find_predecessor(id, ip)
}

func (wrapper *RPC_wrapper) Get_successor(_ Null, ip *string) error {
	return wrapper.node.Get_successor(ip)
}

func (wrapper *RPC_wrapper) Get_predecessor(_ Null, ip *string) error {
	return wrapper.node.Get_predecessor(ip)
}

func (wrapper *RPC_wrapper) Notifty(ip string, _ *Null) error {
	return wrapper.node.Notifty(ip)
}

func (wrapper *RPC_wrapper) Change_predecessor(ip string, _ *Null) error {
	wrapper.node.Change_predecessor(ip)
	return nil
}

func (wrapper *RPC_wrapper) Change_successor(ip string, _ *Null) error {
	wrapper.node.Change_successor(ip)
	return nil
}

func (wrapper *RPC_wrapper) Ping(_ Null, _ *Null) error {
	if wrapper.node.Online {
		return nil
	}
	return errors.New("Offline node.")
}
