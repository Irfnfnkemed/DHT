package chord

import (
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

func (wrapper *RPC_wrapper) Get_successor(_ struct{}, ip *string) error {
	return wrapper.node.Get_successor(ip)
}
