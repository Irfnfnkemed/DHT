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

// type Addr struct {
// 	IP string
// 	ID *big.Int
// }

type Null struct{}

type Node struct {
	Online      bool
	IP          string
	ID          *big.Int
	Predecessor string
	Pre_lock    sync.RWMutex
	Finger      [161]string
	Finger_lock sync.RWMutex
	Server      *rpc.Server
	Listener    net.Listener
	fix_index   int
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
	node.Predecessor = ""
	node.Server = nil
	node.Listener = nil
	node.IP = ip
	node.ID = get_hash(ip)
	node.fix_index = 2
	for i := range node.Finger {
		node.Finger[i] = node.IP
	}
	return true
}

func (node *Node) Create() bool {
	rand.Seed(time.Now().UnixNano())
	logrus.Infof("Create a new DHT net.")
	logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.IP, node.ID)
	node.Pre_lock.Lock()
	node.Predecessor = node.IP
	node.Pre_lock.Unlock()
	node.miantain()
	return true
}

func (node *Node) Find_successor(id *big.Int, ip *string) error {
	pre := ""
	err := node.Find_predecessor(id, &pre)
	if err != nil {
		logrus.Errorf("Find_successor error (IP = %s): %v.", node.IP, err)
		return err
	}
	err = Remote_call(pre, "DHT.Get_successor", Null{}, ip)
	if err != nil {
		logrus.Errorf("Find_successor error (IP = %s): %v.", node.IP, err)
		return err
	}
	return nil
}

func (node *Node) Find_predecessor(id *big.Int, ip *string) error {
	*ip = node.IP
	node.Finger_lock.RLock()
	successor_id := get_hash(node.Finger[1])
	node.Finger_lock.RUnlock()
	if !belong(false, true, node.ID, successor_id, id) {
		err := Remote_call(node.closest_preceding_finger(id), "DHT.Find_predecessor", id, ip)
		if err != nil {
			logrus.Errorf("Find_predecessor error (IP = %s): %v.", node.IP, err)
			return err
		}
	}
	return nil
}

func (node *Node) Get_successor(ip *string) error {
	node.Finger_lock.RLock()
	*ip = node.Finger[1]
	node.Finger_lock.RUnlock()
	return nil
}

func (node *Node) Get_predecessor(ip *string) error {
	node.Pre_lock.RLock()
	*ip = node.Predecessor
	node.Pre_lock.RUnlock()
	return nil
}

func (node *Node) closest_preceding_finger(id *big.Int) string {
	node.Finger_lock.RLock()
	defer node.Finger_lock.RUnlock()
	for i := 160; i > 0; i-- {
		if belong(false, false, node.ID, id, get_hash(node.Finger[i])) {
			return node.Finger[i]
		}
	}
	return node.IP
}

func belong(left_open, right_open bool, beg, end, tar *big.Int) bool {
	cmp_beg_end, cmp_tar_beg, cmp_tar_end := beg.Cmp(end), tar.Cmp(beg), tar.Cmp(end)
	if cmp_beg_end == -1 {
		if cmp_tar_beg == -1 || cmp_tar_end == 1 {
			return false
		} else if cmp_tar_beg == 1 && cmp_tar_end == -1 {
			return true
		} else if cmp_tar_beg == 0 {
			return left_open
		} else if cmp_tar_end == 0 {
			return right_open
		}
	} else if cmp_beg_end == 1 {
		if cmp_tar_beg == -1 && cmp_tar_end == 1 {
			return false
		} else if cmp_tar_beg == 1 || cmp_tar_end == -1 {
			return true
		} else if cmp_tar_beg == 0 {
			return left_open
		} else if cmp_tar_end == 0 {
			return right_open
		}
	} else if cmp_beg_end == 0 { //两端点重合
		if cmp_tar_beg == 0 {
			return left_open || right_open
		} else {
			return true
		}
	}
	return false
}

func (node *Node) Join(ip string) bool {
	logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.IP, node.ID)
	node.Pre_lock.Lock()
	node.Predecessor = ""
	node.Pre_lock.Unlock()
	successor := ""
	err := Remote_call(ip, "DHT.Find_successor", node.ID, &successor)
	if err != nil {
		logrus.Errorf("Join error (IP = %s): %v.", node.IP, err)
		return false
	}
	node.Finger_lock.Lock()
	node.Finger[1] = successor
	node.Finger_lock.Unlock()
	node.miantain()
	return true
}

func (node *Node) stabilize() error {
	node.Finger_lock.RLock()
	successor := node.Finger[1]
	node.Finger_lock.RUnlock()
	ip := ""
	err := Remote_call(successor, "DHT.Get_predecessor", Null{}, &ip)
	if err != nil {
		logrus.Errorf("Stabilize error (IP = %s): %v.", node.IP, err)
		return err
	}
	node.Finger_lock.Lock()
	if (successor == node.IP) ||
		(ip != "" && belong(false, false, node.ID, get_hash(node.Finger[1]), get_hash(ip))) {
		node.Finger[1] = ip
	}
	successor = node.Finger[1]
	node.Finger_lock.Unlock()
	err = Remote_call(successor, "DHT.Notifty", node.IP, &Null{})
	if err != nil {
		logrus.Errorf("Stabilize error (IP = %s): %v.", node.IP, err)
		return err
	}
	return nil
}

func (node *Node) Notifty(ip string) error {
	node.Pre_lock.Lock()
	if node.Predecessor == "" || belong(false, false, get_hash(node.Predecessor), node.ID, get_hash(ip)) {
		node.Predecessor = ip
	}
	node.Pre_lock.Unlock()
	return nil
}

func (node *Node) Run() error {
	node.Online = true
	err := node.Serve()
	if err != nil {
		logrus.Errorf("Run error (IP = %s): %v.", node.IP, err)
		return err
	}
	return nil
}

func (node *Node) miantain() {
	go func() {
		for node.Online {
			node.stabilize()
			time.Sleep(200 * time.Microsecond)
		}
	}()
	go func() {
		for node.Online {
			node.fix_finger()
			time.Sleep(200 * time.Microsecond)
		}
	}()
}

func (node *Node) fix_finger() error {
	ip := ""
	err := node.Find_successor(cal(node.ID, node.fix_index-1), &ip)
	if err != nil {
		logrus.Errorf("Fix_finger error (IP = %s): %v.", node.IP, err)
		return err
	}
	node.Finger_lock.Lock()
	if node.Finger[node.fix_index] != ip { //更新Finger
		node.Finger[node.fix_index] = ip
	}
	node.Finger_lock.Unlock()
	node.fix_index = (node.fix_index-1)%159 + 2
	return nil
}

// 计算n+2^i并取模
func cal(n *big.Int, i int) *big.Int {
	tmp := new(big.Int).Add(n, new(big.Int).Lsh(big.NewInt(1), uint(i)))
	max := new(big.Int).Lsh(big.NewInt(1), uint(160)) //2^160
	if tmp.Cmp(max) >= 0 {
		tmp.Sub(tmp, max)
	}
	return tmp
}

func (node *Node) Change_predecessor(ip string) {
	node.Pre_lock.Lock()
	node.Predecessor = ip
	node.Pre_lock.Unlock()
}

func (node *Node) Change_successor(ip string) {
	node.Finger_lock.Lock()
	node.Finger[1] = ip
	node.Finger_lock.Unlock()
}

func (node *Node) Quit() {
	node.Pre_lock.RLock()
	predecessor := node.Predecessor
	node.Pre_lock.RUnlock()
	node.Finger_lock.RLock()
	successor := node.Finger[1]
	node.Finger_lock.RUnlock()
	go Remote_call(predecessor, "DHT.Change_successor", successor, &Null{})
	go Remote_call(successor, "DHT.Change_predecessor", predecessor, &Null{})
	node.Online = false
	logrus.Infof("Node (IP = %s, ID = %v) quits.", node.IP, node.ID)
}
