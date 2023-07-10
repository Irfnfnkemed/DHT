package chord

import (
	"crypto/sha1"
	"math/big"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Null struct{}

type Node struct {
	Online         bool
	RPC            Node_rpc
	IP             string
	ID             *big.Int
	Predecessor    string
	Pre_lock       sync.RWMutex
	Successor_list [3]string
	Suc_lock       sync.RWMutex
	Finger         [161]string
	Finger_lock    sync.RWMutex
	fix_index      int
	quit           chan bool
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
	node.quit = make(chan bool, 1)
	node.Predecessor = ""
	node.IP = ip
	node.ID = get_hash(ip)
	node.fix_index = 2
	node.Finger_lock.Lock()
	for i := range node.Finger {
		node.Finger[i] = node.IP
	}
	node.Finger_lock.Unlock()
	node.Suc_lock.Lock()
	for i := range node.Successor_list {
		node.Successor_list[i] = node.IP
	}
	node.Suc_lock.Unlock()
	return true
}

func (node *Node) Create() bool {
	rand.Seed(time.Now().UnixNano())
	logrus.Infof("Create a new DHT net.")
	logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.IP, node.ID)
	node.Pre_lock.Lock()
	node.Predecessor = node.IP
	node.Pre_lock.Unlock()
	node.maintain()
	return true
}

func (node *Node) Find_successor(id *big.Int) (ip string, err error) {
	if id.Cmp(node.ID) == 0 { //当前节点即位目标后继，结束
		ip = node.IP
		return ip, nil
	}
	pre, err := node.Find_predecessor(id)
	if err != nil {
		logrus.Errorf("Find_successor error (IP = %s): %v.", node.IP, err)
		return "", err
	}
	err = Remote_call(pre, "DHT.Get_successor", Null{}, &ip)
	if err != nil {
		logrus.Errorf("Find_successor error (IP = %s): %v.", node.IP, err)
		return "", err
	}
	return ip, nil
}

func (node *Node) Find_predecessor(id *big.Int) (ip string, err error) {
	ip = node.IP
	node.Suc_lock.RLock()
	successor_id := get_hash(node.Successor_list[0])
	node.Suc_lock.RUnlock()
	if !belong(false, true, node.ID, successor_id, id) {
		err = Remote_call(node.closest_preceding_finger(id), "DHT.Find_predecessor", id, &ip)
		if err != nil {
			logrus.Errorf("Find_predecessor error (IP = %s): %v.", node.IP, err)
			return "", err
		}
	}
	return ip, nil
}

func (node *Node) Get_successor() (ip string, err error) {
	node.update_successor_list()
	node.Suc_lock.Lock()
	defer node.Suc_lock.Unlock()
	return node.Successor_list[0], nil
}

func (node *Node) Get_predecessor() (ip string, err error) {
	node.Pre_lock.Lock()
	if node.Predecessor == "" || !Ping(node.Predecessor) {
		node.Predecessor = "" //若前驱已下线，置为空
	}
	ip = node.Predecessor
	node.Pre_lock.Unlock()
	return ip, nil
}

func (node *Node) closest_preceding_finger(id *big.Int) string {
	node.Finger_lock.Lock()
	defer node.Finger_lock.Unlock()
	for i := 160; i > 0; i-- {
		if !Ping(node.Finger[i]) { //已下线，将Finger设为仍然上线的位置
			if i == 160 {
				node.Finger[i] = node.IP
			} else {
				node.Finger[i] = node.Finger[i+1]
			}
		} else if belong(false, false, node.ID, id, get_hash(node.Finger[i])) {
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
	node.Suc_lock.Lock()
	node.Successor_list[0] = successor
	node.Suc_lock.Unlock()
	node.maintain()
	return true
}

func (node *Node) stabilize() error {
	successor, err := node.Get_successor()
	if err != nil {
		logrus.Errorf("Stabilize error (IP = %s): %v.", node.IP, err)
		return err
	}
	ip := ""
	err = Remote_call(successor, "DHT.Get_predecessor", Null{}, &ip)
	if err != nil {
		logrus.Errorf("Stabilize error (IP = %s): %v.", node.IP, err)
		return err
	}
	node.Suc_lock.Lock()
	if (successor == node.IP) ||
		(ip != "" && belong(false, false, node.ID, get_hash(successor), get_hash(ip))) {
		node.Successor_list[0] = ip
	}
	successor = node.Successor_list[0]
	node.Suc_lock.Unlock()
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

func (node *Node) maintain() {
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
	ip, err := node.Find_successor(cal(node.ID, node.fix_index-1))
	if err != nil {
		logrus.Errorf("Fixing finger error (IP = %s): %v.", node.IP, err)
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

func (node *Node) Change_successor_list(list [3]string) {
	node.Suc_lock.Lock()
	for i := 1; i < 3; i++ { //将后继列表转移
		node.Successor_list[i] = list[i-1]
	}
	node.Suc_lock.Unlock()
}

func (node *Node) Quit() {
	node.Pre_lock.RLock()
	predecessor := node.Predecessor
	node.Pre_lock.RUnlock()
	node.Suc_lock.RLock()
	successor_list := node.Successor_list
	node.Suc_lock.RUnlock()
	online_successor, err := node.Get_successor()
	if err != nil {
		logrus.Errorf("Quiting error (IP = %s): %v.", node.IP, err)
	}
	if predecessor != "" {
		Remote_call(predecessor, "DHT.Change_successor_list", successor_list, &Null{})
	}
	if err == nil {
		Remote_call(online_successor, "DHT.Change_predecessor", predecessor, &Null{})
	}
	node.Online = false
	close(node.quit)
	logrus.Infof("Node (IP = %s, ID = %v) quits.", node.IP, node.ID)
}

func (node *Node) Get_successor_list() [3]string {
	var successor_list [3]string
	node.Suc_lock.RLock()
	for i := 0; i < 3; i++ {
		successor_list[i] = node.Successor_list[i]
	}
	node.Suc_lock.RUnlock()
	return successor_list
}

func (node *Node) update_successor_list() error {
	var tmp, next_successor_list [3]string
	node.Suc_lock.RLock()
	for i, ip := range node.Successor_list {
		tmp[i] = ip
	}
	node.Suc_lock.RUnlock()
	for _, ip := range tmp {
		if Ping(ip) { //找到最近的存在的后继
			err := Remote_call(ip, "DHT.Get_successor_list", Null{}, &next_successor_list)
			if err != nil {
				logrus.Errorf("Getting successor list error (IP = %s): %v.", node.IP, err)
				continue
			}
			node.Suc_lock.Lock()
			node.Successor_list[0] = ip //更新后继列表
			for j := 1; j < 3; j++ {
				node.Successor_list[j] = next_successor_list[j-1]
			}
			node.Suc_lock.Unlock()
			return nil
		}
	}
	logrus.Infof("All successors are offline (IP = %s).", node.IP)
	//后继列表中节点均失效，尝试通过路由表寻找以防止环断裂
	ip, err := node.Find_successor(cal(node.ID, 0))
	if err != nil {
		logrus.Errorf("Finding successor list error (IP = %s): %v.", node.IP, err)
	}
	node.Suc_lock.Lock()
	for i := range node.Successor_list {
		if i == 0 && ip != "" {
			node.Successor_list[0] = ip
		} else {
			node.Successor_list[i] = node.IP
		}
	}
	node.Suc_lock.Unlock()
	return nil
}

func (node *Node) ForceQuit() {
	node.Online = false
	close(node.quit)
	logrus.Infof("Node (IP = %s, ID = %v) force quits.", node.IP, node.ID)
}
