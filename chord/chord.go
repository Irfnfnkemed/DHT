package chord

import (
	"crypto/sha1"
	"math/big"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"

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

func init() {
	f, _ := os.Create("dht-test.log")
	logrus.SetOutput(f)
}

func (node *Node) Init(ip string) bool {
	node.Online = false
	node.Predecessor = Addr{"", nil}
	node.Server = nil
	node.Listener = nil
	node.This.IP = ip
	node.This.ID = get_hash(ip)
	return true
}

func (node *Node) Create() bool {
	rand.Seed(time.Now().UnixNano())
	node.Finger_lock.Lock()
	for i := range node.Finger {
		node.Finger[i] = node.This
	}
	node.Finger_lock.Unlock()
	logrus.Infof("Create a new DHT net.")
	logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.This.IP, node.This.ID)
	node.miantain()
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
	node.Finger_lock.RLock()
	successor := node.Finger[1].ID
	node.Finger_lock.RUnlock()
	if !belong(false, true, node.This.ID, successor, id) {
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
	node.Finger_lock.RUnlock()
	return nil
}

func (node *Node) Get_predecessor(addr *Addr) error {
	node.Pre_lock.RLock()
	*addr = node.Predecessor
	node.Pre_lock.RUnlock()
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
	logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.This.IP, node.This.ID)
	node.Pre_lock.Lock()
	node.Predecessor = Addr{"", nil}
	node.Pre_lock.Unlock()
	successor := Addr{"", nil}
	err := Remote_call(ip, "DHT.Find_successor", node.This.ID, &successor)
	if err != nil {
		logrus.Errorf("Join error (IP = %s): %v.", node.This.IP, err)
		return false
	}
	node.Finger_lock.Lock()
	node.Finger[1] = successor
	node.Finger_lock.Unlock()
	node.miantain()
	return true
}

func (node *Node) stabilize() error {
	addr := Addr{"", nil}
	node.Finger_lock.RLock()
	successor := node.Finger[1].IP
	node.Finger_lock.RUnlock()
	err := Remote_call(successor, "DHT.Get_successor", Null{}, &addr)
	if err != nil {
		logrus.Errorf("Stabilize error (IP = %s): %v.", node.This.IP, err)
		return err
	}
	node.Finger_lock.Lock()
	if addr.ID != nil && belong(false, false, node.This.ID, node.Finger[1].ID, addr.ID) {
		node.Finger[1] = addr
	}
	successor = node.Finger[1].IP
	node.Finger_lock.Unlock()
	err = Remote_call(successor, "DHT.Notifty", node.This, &Null{})
	if err != nil {
		logrus.Errorf("Stabilize error (IP = %s): %v.", node.This.IP, err)
		return err
	}
	return nil
}

func (node *Node) Notifty(addr Addr) error {
	node.Pre_lock.Lock()
	if node.Predecessor.ID == nil || belong(false, false, node.Predecessor.ID, node.This.ID, addr.ID) {
		node.Predecessor = addr
	}
	node.Pre_lock.Unlock()
	return nil
}

func (node *Node) Run() error {
	defer func() { node.Online = true }()
	err := node.Serve()
	if err != nil {
		logrus.Errorf("Run error (IP = %s): %v.", node.This.IP, err)
		return err
	}
	return nil
}

func (node *Node) miantain() {
	go func() {
		for {
			node.stabilize()
			time.Sleep(200 * time.Millisecond)
		}
	}()
	go func() {
		for {
			node.fix_finger()
			time.Sleep(200 * time.Millisecond)
		}
	}()
}

func (node *Node) fix_finger() error {
	i := rand.Intn(160) + 1 //生成[1,160]范围内的整数
	ip := ""
	err := Remote_call(node.This.IP, "DHT.Find_successor", cal(node.This.ID, i-1), &ip)
	if err != nil {
		logrus.Errorf("Fix_finger error (IP = %s): %v.", node.This.IP, err)
		return err
	}
	node.Finger_lock.Lock()
	if node.Finger[i].IP != ip { //更新Finger
		node.Finger[i] = Addr{ip, get_hash(ip)}
	}
	node.Finger_lock.Unlock()
	return nil
}

// 计算n+2^i
func cal(n *big.Int, i int) *big.Int {
	return new(big.Int).Add(n, new(big.Int).Lsh(big.NewInt(1), uint(i)))
}
