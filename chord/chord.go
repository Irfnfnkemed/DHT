package chord

import (
	"crypto/sha1"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Null struct{}

type Value_pair struct {
	Value  string
	Key_id *big.Int
}

type Data_pair struct {
	Key        string
	Value_pair Value_pair
}

type IP_pair struct {
	IP_pre     string
	IP_pre_pre string
}

type Node struct {
	Online           bool
	RPC              Node_rpc
	IP               string
	ID               *big.Int
	Predecessor      string
	Pre_lock         sync.RWMutex
	Successor_list   [3]string
	Suc_lock         sync.RWMutex
	Finger           [161]string
	Finger_lock      sync.RWMutex
	data             map[string]Value_pair
	data_lock        sync.RWMutex
	data_backup      map[string]Value_pair
	data_backup_lock sync.RWMutex
	fix_index        int
	start            chan bool
	quit             chan bool
}

func (node *Node) B() {
	fmt.Println("data:", node.data)
	fmt.Println("backup:", node.data_backup)
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
	node.start = make(chan bool, 1)
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
	node.data_lock.Lock()
	node.data = make(map[string]Value_pair)
	node.data_lock.Unlock()
	node.data_backup_lock.Lock()
	node.data_backup = make(map[string]Value_pair)
	node.data_backup_lock.Unlock()
	return true
}

func (node *Node) Create() {
	rand.Seed(time.Now().UnixNano())
	logrus.Infof("Create a new DHT net.")
	logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.IP, node.ID)
	node.Pre_lock.Lock()
	node.Predecessor = node.IP
	node.Pre_lock.Unlock()
	node.maintain()
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
	select { // 阻塞直至run()完成
	case <-node.start:
		err := Remote_call(ip, "DHT.Find_successor", node.ID, &successor)
		if err != nil {
			logrus.Errorf("Join error (IP = %s): %v.", node.IP, err)
			return false
		}
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
	ok := ((ip != "") && belong(false, false, node.ID, get_hash(successor), get_hash(ip))) //是否需要更改后继
	if (successor == node.IP) || ok {
		node.Suc_lock.Lock()
		node.Successor_list[2] = node.Successor_list[1]
		node.Successor_list[1] = node.Successor_list[0]
		node.Successor_list[0] = ip
		node.Suc_lock.Unlock()
		Remote_call(successor, "DHT.Transfer_data", IP_pair{ip, node.IP}, &Null{})
	}
	node.Suc_lock.RLock()
	successor = node.Successor_list[0]
	node.Suc_lock.RUnlock()

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

func (node *Node) Run() {
	node.Online = true
	go node.Serve()
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
	for i := 0; i < 3; i++ { //将后继列表转移
		node.Successor_list[i] = list[i]
	}
	node.Suc_lock.Unlock()
}

func (node *Node) Quit() {
	if !node.Online {
		return
	}
	node.Online = false
	close(node.quit)
	node.Pre_lock.RLock()
	predecessor := node.Predecessor
	node.Pre_lock.RUnlock()
	node.update_successor_list()
	var successor_list [3]string
	node.Suc_lock.RLock()
	successor_list[0] = node.Successor_list[0]
	successor_list[1] = node.Successor_list[1]
	successor_list[2] = node.Successor_list[2]
	node.Suc_lock.RUnlock()
	if predecessor != "" {
		Remote_call(predecessor, "DHT.Change_successor_list", successor_list, &Null{})
	}
	Remote_call(successor_list[0], "DHT.Change_predecessor", predecessor, &Null{})
	go func() { //转移主数据
		node.data_lock.RLock()
		data := []Data_pair{}
		for key, value_pair := range node.data {
			data = append(data, Data_pair{key, value_pair})
		}
		node.data_lock.RUnlock()
		Remote_call(successor_list[0], "DHT.Put_in_all", data, &Null{})
		Remote_call(successor_list[0], "DHT.Delete_off_backup", data, &Null{})
	}()
	go func() { //转移备份
		node.data_backup_lock.RLock()
		data_backup := []Data_pair{}
		for key, value_pair := range node.data_backup {
			data_backup = append(data_backup, Data_pair{key, value_pair})
		}
		node.data_backup_lock.RUnlock()
		Remote_call(successor_list[0], "DHT.Put_in_backup", data_backup, &Null{})
	}()
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

func (node *Node) Put(key string, value string) bool {
	id := get_hash(key)
	ip, _ := node.Find_successor(id)
	err := Remote_call(ip, "DHT.Put_in_all", []Data_pair{{key, Value_pair{value, id}}}, &Null{})
	if err != nil {
		logrus.Errorf("Node (IP = %s) putting in data error (key = %s, value = %s): %v.", node.IP, key, value, err)
		return false
	}
	logrus.Infof("Node (IP = %s) puts in data : key = %s, value = %s.", node.IP, key, value)
	return true
}

func (node *Node) Put_in(data []Data_pair) error {
	node.data_lock.Lock()
	for i := range data {
		node.data[data[i].Key] = data[i].Value_pair
	}
	node.data_lock.Unlock()
	return nil
}

func (node *Node) Put_in_all(data []Data_pair) error {
	node.Put_in(data)
	//后继节点放入备份数据
	successor, err := node.Get_successor()
	if err != nil {
		logrus.Errorf("Getting successor error (IP = %s): %v.", node.IP, err)
		return err
	}
	err = Remote_call(successor, "DHT.Put_in_backup", data, &Null{})
	if err != nil {
		logrus.Errorf("Putting in backup error (IP = %s): %v.", node.IP, err)
	}
	return nil
}

func (node *Node) Put_in_backup(data []Data_pair) error {
	node.data_backup_lock.Lock()
	for i := range data {
		node.data_backup[data[i].Key] = data[i].Value_pair
	}
	node.data_backup_lock.Unlock()
	return nil
}

func (node *Node) Get(key string) (ok bool, value string) {
	id := get_hash(key)
	node.Pre_lock.RLock()
	pre := node.Predecessor
	node.Pre_lock.RUnlock()
	if belong(false, true, get_hash(pre), node.ID, id) {
		value, ok = node.Get_out(key)
	} else {
		ip, _ := node.Find_successor(id)
		err := Remote_call(ip, "DHT.Get_out", key, &value)
		if err != nil {
			logrus.Errorf("Node (IP = %s) getting out data error (key = %s, value = %s): %v.", node.IP, key, value, err)
			return false, ""
		}
	}
	logrus.Infof("Node (IP = %s) gets out data : key = %s, value = %s.", node.IP, key, value)
	return true, value
}

func (node *Node) Get_out(key string) (string, bool) {
	node.data_lock.RLock()
	defer node.data_lock.RUnlock()
	value_pair, ok := node.data[key]
	return value_pair.Value, ok
}

func (node *Node) Transfer_data(ips IP_pair) error { //将数据转移到前节点
	pre_id := get_hash(ips.IP_pre)
	pre_pre_id := get_hash(ips.IP_pre_pre)
	//转移主数据
	data := []Data_pair{}
	node.data_lock.RLock()
	for key, value_pair := range node.data {
		if !belong(false, true, pre_id, node.ID, value_pair.Key_id) {
			data = append(data, Data_pair{key, value_pair})
		}
	}
	node.data_lock.RUnlock()
	node.data_lock.Lock()
	for _, data_pair := range data {
		delete(node.data, data_pair.Key)
	}
	node.data_lock.Unlock()
	err := Remote_call(ips.IP_pre, "DHT.Put_in", data, &Null{})
	if err != nil {
		logrus.Errorf("Node (IP = %s , to_ip = %s) transferring data error: %v.", node.IP, ips.IP_pre, err)
		return err
	}
	//转移主数据对应的备份
	key_backup := []string{}
	node.data_backup_lock.Lock()
	for _, data_pair := range data {
		node.data_backup[data_pair.Key] = data_pair.Value_pair
		key_backup = append(key_backup, data_pair.Key)
	}
	node.data_backup_lock.Unlock()
	successor, _ := node.Get_successor()
	err = Remote_call(successor, "DHT.Delete_off_backup", key_backup, &Null{})
	if err != nil {
		logrus.Errorf("Node (IP = %s , suc_ip = %s) transferring backup data error: %v.", node.IP, successor, err)
		return err
	}
	//转移备份
	data_backup := []Data_pair{}
	node.data_backup_lock.RLock()
	for key, value_pair := range node.data_backup {
		if !belong(false, true, pre_pre_id, pre_id, value_pair.Key_id) {
			data_backup = append(data_backup, Data_pair{key, value_pair})
		}
	}
	node.data_backup_lock.RUnlock()
	node.data_backup_lock.Lock()
	for _, data_pair := range data_backup {
		delete(node.data_backup, data_pair.Key)
	}
	node.data_backup_lock.Unlock()
	err = Remote_call(ips.IP_pre, "DHT.Put_in_backup", data_backup, &Null{})
	if err != nil {
		logrus.Errorf("Node (IP = %s , to_ip = %s) transferring backup data error: %v.", node.IP, ips.IP_pre, err)
		return err
	}
	return nil
}

func (node *Node) Delete(key string) bool {
	id := get_hash(key)
	ip, _ := node.Find_successor(id)
	err := Remote_call(ip, "DHT.Delete_off", []string{key}, &Null{})
	if err != nil {
		logrus.Errorf("Node (IP = %s) deleting off data error (key = %s): %v.", node.IP, key, err)
		return false
	}
	logrus.Infof("Node (IP = %s) delete off data : key = %s.", node.IP, key)
	return true
}

func (node *Node) Delete_off(keys []string) bool {
	out := true
	node.data_lock.Lock()
	for _, key := range keys {
		_, ok := node.data[key]
		if !ok {
			out = false
		} else {
			delete(node.data, key)
		}
	}
	node.data_lock.Unlock()
	//删除后继节点的备份数据
	successor, err := node.Get_successor()
	if err != nil {
		logrus.Errorf("Getting successor error (IP = %s): %v.", node.IP, err)
		return false
	}
	err = Remote_call(successor, "DHT.Delete_off_backup", keys, &Null{})
	if err != nil {
		logrus.Errorf("Deleting off backup error (IP = %s): %v.", node.IP, err)
		return false
	}
	return out
}

func (node *Node) Delete_off_backup(keys []string) bool {
	out := true
	node.data_backup_lock.Lock()
	for _, key := range keys {
		_, ok := node.data_backup[key]
		if !ok {
			out = false
		} else {
			delete(node.data_backup, key)
		}
	}
	node.data_backup_lock.Unlock()
	return out
}
func (node *Node) A() {

	fmt.Println(node.IP, node.Predecessor, node.Successor_list[0])

	fmt.Println(node.ID)

	fmt.Println(node.Successor_list)

	fmt.Println(Ping(node.IP))
}
