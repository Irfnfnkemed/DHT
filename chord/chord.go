package chord

import (
	"crypto/sha1"
	"math/big"
	"net"
	"net/rpc"
	"sync"

	"github.com/sirupsen/logrus"
)

type Addr struct {
	IP string
	ID *big.Int
}

type Null struct{}

type Node struct {
	Online      bool
	This        Addr
	Predecessor Addr
	Pre_lock    sync.RWMutex
	Finger      [161]Addr
	Finger_lock sync.RWMutex
	Server      *rpc.Server
	Listener    net.Listener
}

func get_hash(Addr_IP string) *big.Int {
	hash := sha1.Sum([]byte(Addr_IP))
	hashInt := new(big.Int)
	return hashInt.SetBytes(hash[:])
}

func (node *Node) Init(ip string) bool {
	node.Online = false
	node.This.IP = ip
	node.This.ID = get_hash(ip)
	return true
}

func (node *Node) Create() bool {
	defer func() { node.Online = true }()
	node.Finger_lock.Lock()
	for i := range node.Finger {
		node.Finger[i] = node.This
	}
	logrus.Infof("Create a new DHT net.")
	logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.This.IP, node.This.ID)
	defer node.Finger_lock.Unlock()
	return true
}

func (node *Node) Find_successor(id *big.Int, ip *string) error {
	pre_ip := ""
	err := node.Find_predecessor(id, &pre_ip)
	if err != nil {
		logrus.Errorf("Find_successor error (IP = %s): %v.", node.This.IP, err)
		return err
	}
	err = Remote_call(pre_ip, "DHT.Get_successor", Null{}, &ip)
	if err != nil {
		logrus.Errorf("Find_successor error (IP = %s): %v.", node.This.IP, err)
		return err
	}
	return nil
}

func (node *Node) Find_predecessor(id *big.Int, ip *string) error {
	*ip = node.This.IP
	if !belong(false, true, node.This.ID, node.Finger[1].ID, id) {
		err := Remote_call(node.closest_preceding_finger(id), "DHT.Find_predecessor", id, &ip)
		if err != nil {
			logrus.Errorf("Find_predecessor error (IP = %s): %v.", node.This.IP, err)
			return err
		}
	}
	return nil
}

func (node *Node) Get_successor(addr *Addr) error {
	node.Finger_lock.RLock()
	*addr = node.Finger[1]
	defer node.Finger_lock.RUnlock()
	return nil
}

func (node *Node) Get_predecessor(addr *Addr) error {
	node.Pre_lock.RLock()
	*addr = node.Predecessor
	defer node.Pre_lock.RUnlock()
	return nil
}

func (node *Node) closest_preceding_finger(id *big.Int) string {
	node.Finger_lock.RLock()
	for i := 160; i > 0; i-- {
		if belong(false, false, node.This.ID, node.Finger[i].ID, id) {
			return node.Finger[i].IP
		}
	}
	defer node.Finger_lock.RUnlock()
	return node.This.IP
}

func belong(left_open, right_open bool, beg, end, tar *big.Int) bool {
	cmp_beg_end, cmp_tar_beg, cmp_tar_end := beg.Cmp(end), tar.Cmp(beg), tar.Cmp(end)
	if cmp_tar_beg == 0 {
		return left_open
	} else if cmp_tar_end == 0 {
		return right_open
	} else if cmp_beg_end == -1 {
		if cmp_tar_beg == -1 || cmp_tar_end == 1 {
			return false
		} else if cmp_tar_beg == 1 && cmp_tar_end == -1 {
			return true
		}
	} else if cmp_beg_end == 1 {
		if cmp_tar_beg == -1 && cmp_tar_end == 1 {
			return false
		} else if cmp_tar_beg == 1 || cmp_tar_end == -1 {
			return true
		}
	} else if cmp_beg_end == 0 {
		return left_open && right_open && cmp_tar_beg == 0
	}
	return false
}

func (node *Node) Join(ip string) bool {
	defer func() { node.Online = true }()
	defer node.Finger_lock.Unlock()
	defer node.Pre_lock.Unlock()
	logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.This.IP, node.This.ID)
	node.Pre_lock.Lock()
	node.Predecessor = Addr{"", nil}
	node.Finger_lock.Lock()
	err := Remote_call(ip, "DHT.Find_successor", node.This.ID, &node.Finger[1])
	if err != nil {
		logrus.Errorf("Join error (IP = %s): %v.", node.This.IP, err)
		return false
	}
	return true
}

func (node *Node) stabilize() error {
	addr := Addr{"", nil}
	node.Finger_lock.Lock()
	defer node.Finger_lock.Unlock()
	err := Remote_call(node.Finger[1].IP, "DHT.Get_successor", Null{}, &addr)
	if err != nil {
		logrus.Errorf("Stabilize error (IP = %s): %v.", node.This.IP, err)
		return err
	}
	if addr.ID != nil && belong(false, false, node.This.ID, node.Finger[1].ID, addr.ID) {
		node.Finger[1] = addr
	}
	err = Remote_call(node.Finger[1].IP, "DHT.Notifty", node.This, Null{})
	if err != nil {
		logrus.Errorf("Stabilize error (IP = %s): %v.", node.This.IP, err)
		return err
	}
	return nil
}

func (node *Node) Notifty(addr Addr) error {
	node.Finger_lock.Lock()
	defer node.Finger_lock.Unlock()
	if node.Predecessor.ID == nil || belong(false, false, node.Predecessor.ID, node.This.ID, addr.ID) {
		node.Predecessor = addr
	}
	return nil
}
