package chord

import (
	"errors"
	"math/big"
)

type RPC_wrapper struct {
	node *Node
}

func (wrapper *RPC_wrapper) Find_successor(id *big.Int, ip *string) (err error) {
	*ip, err = wrapper.node.Find_successor(id)
	return err
}

func (wrapper *RPC_wrapper) Find_predecessor(id *big.Int, ip *string) (err error) {
	*ip, err = wrapper.node.Find_predecessor(id)
	return err
}

func (wrapper *RPC_wrapper) Get_successor(_ Null, ip *string) (err error) {
	*ip, err = wrapper.node.Get_successor()
	return err
}

func (wrapper *RPC_wrapper) Get_predecessor(_ Null, ip *string) (err error) {
	*ip, err = wrapper.node.Get_predecessor()
	return err
}

func (wrapper *RPC_wrapper) Notifty(ip string, _ *Null) error {
	return wrapper.node.Notifty(ip)
}

func (wrapper *RPC_wrapper) Change_predecessor(ip string, _ *Null) error {
	wrapper.node.Change_predecessor(ip)
	return nil
}

func (wrapper *RPC_wrapper) Change_successor_list(list [3]string, _ *Null) error {
	wrapper.node.Change_successor_list(list)
	return nil
}

func (wrapper *RPC_wrapper) Ping(_ Null, _ *Null) error {
	if wrapper.node.Online {
		return nil
	}
	return errors.New("Offline node.")
}

func (wrapper *RPC_wrapper) Get_successor_list(_ Null, successor_list *[3]string) error {
	*successor_list = wrapper.node.Get_successor_list()
	return nil
}

func (wrapper *RPC_wrapper) Put_in(data data_pair, _ *Null) error {
	return wrapper.node.Put_in(data)
}
