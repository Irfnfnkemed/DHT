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

func (wrapper *RPC_wrapper) Put_in(data []Data_pair, _ *Null) error {
	return wrapper.node.Put_in(data)
}

func (wrapper *RPC_wrapper) Put_in_all(data []Data_pair, _ *Null) error {
	return wrapper.node.Put_in_all(data)
}

func (wrapper *RPC_wrapper) Put_in_backup(data []Data_pair, _ *Null) error {
	return wrapper.node.Put_in_backup(data)
}

func (wrapper *RPC_wrapper) Get_out(key string, value *string) error {
	ok := false
	*value, ok = wrapper.node.Get_out(key)
	if ok {
		return nil
	} else {
		return errors.New("Get out error.")
	}
}

func (wrapper *RPC_wrapper) Transfer_data(ips IP_pair, _ *Null) error {
	return wrapper.node.Transfer_data(ips)
}

func (wrapper *RPC_wrapper) Delete_off_all(keys []string, _ *Null) error {
	ok := wrapper.node.Delete_off_all(keys)
	if ok {
		return nil
	} else {
		return errors.New("Delete off error.")
	}
}

func (wrapper *RPC_wrapper) Delete_off_backup(keys []string, _ *Null) error {
	ok := wrapper.node.Delete_off_backup(keys)
	if ok {
		return nil
	} else {
		return errors.New("Delete off backup error.")
	}
}

func (wrapper *RPC_wrapper) Lock(_Null, _ *Null) error {
	wrapper.node.Lock()
	return nil
}

func (wrapper *RPC_wrapper) Unlock(_Null, _ *Null) error {
	wrapper.node.Unlock()
	return nil
}
