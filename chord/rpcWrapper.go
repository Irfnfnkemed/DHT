package chord

import (
	"errors"
	"math/big"
)

type RPCWrapper struct {
	node *Node
}

func (wrapper *RPCWrapper) FindSuccessor(id *big.Int, ip *string) (err error) {
	*ip, err = wrapper.node.FindSuccessor(id)
	return err
}

func (wrapper *RPCWrapper) FindPredecessor(id *big.Int, ip *string) (err error) {
	*ip, err = wrapper.node.FindPredecessor(id)
	return err
}

func (wrapper *RPCWrapper) GetSuccessor(_ Null, ip *string) (err error) {
	*ip, err = wrapper.node.GetSuccessor()
	return err
}

func (wrapper *RPCWrapper) GetPredecessor(_ Null, ip *string) (err error) {
	*ip, err = wrapper.node.GetPredecessor()
	return err
}

func (wrapper *RPCWrapper) Notifty(ip string, _ *Null) error {
	return wrapper.node.Notifty(ip)
}

func (wrapper *RPCWrapper) ChangePredecessor(ip string, _ *Null) error {
	wrapper.node.ChangePredecessor(ip)
	return nil
}

func (wrapper *RPCWrapper) ChangeSuccessorList(list [3]string, _ *Null) error {
	wrapper.node.ChangeSuccessorList(list)
	return nil
}

func (wrapper *RPCWrapper) Ping(_ Null, _ *Null) error {
	if wrapper.node.Online {
		return nil
	}
	return errors.New("Offline node.")
}

func (wrapper *RPCWrapper) GetSuccessorList(_ Null, successorList *[3]string) error {
	*successorList = wrapper.node.GetSuccessorList()
	return nil
}

func (wrapper *RPCWrapper) PutIn(data []DataPair, _ *Null) error {
	return wrapper.node.PutIn(data)
}

func (wrapper *RPCWrapper) PutInAll(data []DataPair, _ *Null) error {
	return wrapper.node.PutInAll(data)
}

func (wrapper *RPCWrapper) PutInBackup(data []DataPair, _ *Null) error {
	return wrapper.node.PutInBackup(data)
}

func (wrapper *RPCWrapper) GetOut(key string, value *string) error {
	ok := false
	*value, ok = wrapper.node.GetOut(key)
	if ok {
		return nil
	} else {
		return errors.New("Get out error.")
	}
}

func (wrapper *RPCWrapper) TransferData(ips IpPair, _ *Null) error {
	return wrapper.node.TransferData(ips)
}

func (wrapper *RPCWrapper) DeleteOffAll(keys []string, _ *Null) error {
	ok := wrapper.node.DeleteOffAll(keys)
	if ok {
		return nil
	} else {
		return errors.New("Delete off error.")
	}
}

func (wrapper *RPCWrapper) DeleteOffBackup(keys []string, _ *Null) error {
	ok := wrapper.node.DeleteOffBackup(keys)
	if ok {
		return nil
	} else {
		return errors.New("Delete off backup error.")
	}
}

func (wrapper *RPCWrapper) Lock(_Null, _ *Null) error {
	wrapper.node.Lock()
	return nil
}

func (wrapper *RPCWrapper) Unlock(_Null, _ *Null) error {
	wrapper.node.Unlock()
	return nil
}
