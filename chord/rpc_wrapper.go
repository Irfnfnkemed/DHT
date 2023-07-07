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

func (wrapper *RPC_wrapper) Get_successor(_ Null, addr *Addr) error {
	return wrapper.Core_node.Get_successor(addr)
}

func (wrapper *RPC_wrapper) Get_predecessor(_ Null, addr *Addr) error {
	return wrapper.Core_node.Get_predecessor(addr)
}

func (wrapper *RPC_wrapper) Notifty(addr Addr, _ *Null) error {
	return wrapper.Core_node.Notifty(addr)
}
