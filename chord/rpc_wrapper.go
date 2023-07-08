package chord

import (
	"math/big"
)

type RPC_wrapper struct {
	Core_node *Node
}

func (wrapper *RPC_wrapper) Find_successor(id *big.Int, ip *string) error {
	return wrapper.Core_node.Find_successor(id, ip)
}

func (wrapper *RPC_wrapper) Find_predecessor(id *big.Int, ip *string) error {
	return wrapper.Core_node.Find_predecessor(id, ip)
}

func (wrapper *RPC_wrapper) Get_successor(_ Null, ip *string) error {
	return wrapper.Core_node.Get_successor(ip)
}

func (wrapper *RPC_wrapper) Get_predecessor(_ Null, ip *string) error {
	return wrapper.Core_node.Get_predecessor(ip)
}

func (wrapper *RPC_wrapper) Notifty(ip string, _ *Null) error {
	return wrapper.Core_node.Notifty(ip)
}

func (wrapper *RPC_wrapper) Change_predecessor(ip string, _ *Null) error {
	wrapper.Core_node.Change_predecessor(ip)
	return nil
}

func (wrapper *RPC_wrapper) Change_successor(ip string, _ *Null) error {
	wrapper.Core_node.Change_successor(ip)
	return nil
}
