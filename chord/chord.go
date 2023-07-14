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

type Null struct{}

type Node struct {
	Online           bool
	RPC              Node_rpc
	IP               string
	ID               *big.Int
	predecessor      string
	pre_lock         sync.RWMutex
	successor_list   [3]string
	suc_lock         sync.RWMutex
	finger           [161]string
	finger_lock      sync.RWMutex
	data             map[string]Value_pair
	data_lock        sync.RWMutex
	data_backup      map[string]Value_pair
	data_backup_lock sync.RWMutex
	fix_index        int
	finger_start     [161]*big.Int
	block            sync.Mutex
	start            chan bool
	quit             chan bool
}

var exp [161]*big.Int

// 整体初始化
func init() {
	f, _ := os.Create("dht-test.log")
	logrus.SetOutput(f)
	for i := range exp {
		exp[i] = new(big.Int).Lsh(big.NewInt(1), uint(i)) //exp[i]存储2^i
	}
}

// 节点初始化
func (node *Node) Init(ip string) bool {
	node.Online = false
	node.start = make(chan bool, 1)
	node.quit = make(chan bool, 1)
	node.predecessor = ""
	node.IP = ip
	node.ID = get_hash(ip)
	node.fix_index = 2
	for i := 2; i <= 160; i++ {
		node.finger_start[i] = cal(node.ID, i-1)
	}
	node.finger_lock.Lock()
	for i := range node.finger {
		node.finger[i] = node.IP
	}
	node.finger_lock.Unlock()
	node.suc_lock.Lock()
	for i := range node.successor_list {
		node.successor_list[i] = node.IP
	}
	node.suc_lock.Unlock()
	node.data_lock.Lock()
	node.data = make(map[string]Value_pair)
	node.data_lock.Unlock()
	node.data_backup_lock.Lock()
	node.data_backup = make(map[string]Value_pair)
	node.data_backup_lock.Unlock()
	return true
}

// 节点开始运作
func (node *Node) Run() {
	node.Online = true
	go node.Serve()
}

// 创建DHT网络(加入第一个节点)
func (node *Node) Create() {
	rand.Seed(time.Now().UnixNano())
	logrus.Infof("Create a new DHT net.")
	node.pre_lock.Lock()
	node.predecessor = node.IP
	node.pre_lock.Unlock()
	select { // 阻塞直至run()完成
	case <-node.start:
		logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.IP, node.ID)
	}
	node.maintain()
}

// 向DHT网络中加入节点
func (node *Node) Join(ip string) bool {
	node.pre_lock.Lock()
	node.predecessor = ""
	node.pre_lock.Unlock()
	successor := ""
	select { // 阻塞直至run()完成
	case <-node.start:
		logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.IP, node.ID)
	}
	err := Remote_call(ip, "DHT.Find_successor", node.ID, &successor)
	if err != nil {
		logrus.Errorf("Join error (IP = %s): %v.", node.IP, err)
		return false
	}
	node.suc_lock.Lock()
	node.successor_list[0] = successor
	node.suc_lock.Unlock()
	node.maintain()
	return true
}

// 正常退出DHT网络(通知其他节点)
func (node *Node) Quit() {
	if !node.Online {
		return
	}
	node.block.Lock() //阻塞stablize
	node.pre_lock.RLock()
	predecessor := node.predecessor
	node.pre_lock.RUnlock()
	node.update_successor_list()
	var successor_list [3]string
	node.suc_lock.RLock()
	successor_list[0] = node.successor_list[0]
	successor_list[1] = node.successor_list[1]
	successor_list[2] = node.successor_list[2]
	node.suc_lock.RUnlock()
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		if predecessor != "" && predecessor != "OFFLINE" && predecessor != node.IP {
			Remote_call(predecessor, "DHT.Lock", Null{}, &Null{})
		}
		if predecessor != "" && predecessor != "OFFLINE" {
			Remote_call(predecessor, "DHT.Change_successor_list", successor_list, &Null{})
		}
		wg.Done()
	}()
	go func() {
		if predecessor != successor_list[0] && successor_list[0] != node.IP {
			Remote_call(successor_list[0], "DHT.Lock", Null{}, &Null{})
		}
		Remote_call(successor_list[0], "DHT.Change_predecessor", predecessor, &Null{})
		wg.Done()
	}()
	go func() { //转移主数据
		node.data_lock.RLock()
		data := []Data_pair{}
		keys := []string{}
		for key, value_pair := range node.data {
			data = append(data, Data_pair{key, value_pair})
			keys = append(keys, key)
		}
		node.data_lock.RUnlock()
		Remote_call(successor_list[0], "DHT.Put_in_all", data, &Null{})
		Remote_call(successor_list[0], "DHT.Delete_off_backup", keys, &Null{})
		wg.Done()
	}()
	go func() { //转移备份
		node.data_backup_lock.RLock()
		data_backup := []Data_pair{}
		for key, value_pair := range node.data_backup {
			data_backup = append(data_backup, Data_pair{key, value_pair})
		}
		node.data_backup_lock.RUnlock()
		Remote_call(successor_list[0], "DHT.Put_in_backup", data_backup, &Null{})
		wg.Done()
	}()
	wg.Wait()
	if predecessor != "" && predecessor != "OFFLINE" {
		Remote_call(predecessor, "DHT.Unlock", Null{}, &Null{})
	}
	if predecessor != successor_list[0] {
		Remote_call(successor_list[0], "DHT.Unlock", Null{}, &Null{})
	}
	node.Online = false
	close(node.quit)
	logrus.Infof("Node (IP = %s, ID = %v) quits.", node.IP, node.ID)
}

// 强制退出DHT网络(未通知其他节点)
func (node *Node) ForceQuit() {
	if !node.Online {
		return
	}
	node.Online = false
	close(node.quit)
	logrus.Infof("Node (IP = %s, ID = %v) force quits.", node.IP, node.ID)
}

// 存入数据
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

// 查询数据
func (node *Node) Get(key string) (ok bool, value string) {
	id := get_hash(key)
	node.pre_lock.RLock()
	pre := node.predecessor
	node.pre_lock.RUnlock()
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

// 删除数据
func (node *Node) Delete(key string) bool {
	id := get_hash(key)
	ip, _ := node.Find_successor(id)
	err := Remote_call(ip, "DHT.Delete_off_all", []string{key}, &Null{})
	if err != nil {
		logrus.Errorf("Node (IP = %s) deleting off data error (key = %s): %v.", node.IP, key, err)
		return false
	}
	logrus.Infof("Node (IP = %s) delete off data : key = %s.", node.IP, key)
	return true
}

// 找某个地址id的后继节点
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

// 找某个地址id的前驱节点
func (node *Node) Find_predecessor(id *big.Int) (ip string, err error) {
	ip = node.IP
	node.suc_lock.RLock()
	successor_id := get_hash(node.successor_list[0])
	node.suc_lock.RUnlock()
	if !belong(false, true, node.ID, successor_id, id) {
		err = Remote_call(node.closest_preceding_finger(id), "DHT.Find_predecessor", id, &ip)
		if err != nil {
			logrus.Errorf("Find_predecessor error (IP = %s): %v.", node.IP, err)
			return "", err
		}
	}
	return ip, nil
}

// 得到某节点的后继(同时顺带更新successor_list)
func (node *Node) Get_successor() (ip string, err error) {
	node.update_successor_list()
	node.suc_lock.Lock()
	defer node.suc_lock.Unlock()
	return node.successor_list[0], nil
}

// 得到某节点的前驱(同时顺带监测前驱是否异常下线，若发现则利用节点的备份数据进行数据恢复)
func (node *Node) Get_predecessor() (string, error) {
	node.pre_lock.RLock()
	predecessor := node.predecessor
	node.pre_lock.RUnlock()
	if predecessor != "" && !Ping(predecessor) {
		node.pre_lock.RLock()
		node.pre_lock.RUnlock()
		node.pre_lock.Lock()
		node.predecessor = "OFFLINE" //若前驱已下线，置上标记
		predecessor = "OFFLINE"
		node.pre_lock.Unlock()
		data_backup := []Data_pair{}
		node.data_backup_lock.Lock()
		for key, value_pair := range node.data_backup {
			data_backup = append(data_backup, Data_pair{key, value_pair})
		}
		for _, data_pair := range data_backup {
			delete(node.data_backup, data_pair.Key) //从备份中删去
		}
		node.data_backup_lock.Unlock()
		node.data_lock.Lock()
		for _, data_pair := range data_backup {
			node.data[data_pair.Key] = data_pair.Value_pair //加入主数据
		}
		node.data_lock.Unlock()
		successor, err := node.Get_successor()
		if err != nil {
			logrus.Errorf("Restoring data error (IP = %s): %v.", node.IP, err)
			return "OFFLINE", err
		}
		err = Remote_call(successor, "DHT.Put_in_backup", data_backup, &Null{})
		if err != nil {
			logrus.Errorf("Restoring data error (IP = %s): %v.", node.IP, err)
			return "OFFLINE", err
		}
	}
	return predecessor, nil
}

// 找到路由表中在目标位置前最近的上线节点
func (node *Node) closest_preceding_finger(id *big.Int) string {
	for i := 160; i > 1; i-- {
		if !Ping(node.finger[i]) { //已下线，将finger设为仍然上线的位置
			if i == 160 {
				node.finger[i] = node.IP
			} else {
				node.finger[i] = node.finger[i+1]
			}
		} else if belong(false, false, node.ID, id, get_hash(node.finger[i])) {
			return node.finger[i]
		}
	}
	node.suc_lock.Lock()
	defer node.suc_lock.Unlock()
	if !Ping(node.successor_list[0]) { //已下线，将finger设为仍然上线的位置
		node.successor_list[0] = node.IP
	} else if belong(false, false, node.ID, id, get_hash(node.successor_list[0])) {
		return node.successor_list[0]
	}
	return node.IP
}

// 修护前驱后继，并相应地转移数据
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
	if (successor == node.IP) ||
		(ip != "" && ip != "OFFLINE" && belong(false, false, node.ID, get_hash(successor), get_hash(ip))) {
		node.suc_lock.Lock() //需要更改后继
		node.successor_list[2] = node.successor_list[1]
		node.successor_list[1] = node.successor_list[0]
		node.successor_list[0] = ip
		node.suc_lock.Unlock()
		Remote_call(successor, "DHT.Transfer_data", IP_pair{ip, node.IP}, &Null{})
	}
	node.suc_lock.RLock()
	successor = node.successor_list[0]
	node.suc_lock.RUnlock()
	err = Remote_call(successor, "DHT.Notifty", node.IP, &Null{})
	if err != nil {
		logrus.Errorf("Stabilize error (IP = %s): %v.", node.IP, err)
		return err
	}
	if ip == "OFFLINE" {
		data := []Data_pair{}
		node.data_lock.RLock()
		for key, value_pair := range node.data {
			data = append(data, Data_pair{key, value_pair})
		}
		node.data_lock.RUnlock()
		err = Remote_call(successor, "DHT.Put_in_backup", data, &Null{})
		if err != nil {
			logrus.Errorf("Restoring data error (IP = %s): %v.", node.IP, err)
			return err
		}
	}
	return nil
}

// 修复前驱
func (node *Node) Notifty(ip string) error {
	node.pre_lock.Lock()
	if node.predecessor == "" || node.predecessor == "OFFLINE" ||
		belong(false, false, get_hash(node.predecessor), node.ID, get_hash(ip)) {
		node.predecessor = ip
	}
	node.pre_lock.Unlock()
	return nil
}

// 更改后继时，转移数据
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

// 修复路由表
func (node *Node) fix_finger() error {
	ip, err := node.Find_successor(node.finger_start[node.fix_index])
	if err != nil {
		logrus.Errorf("Fixing finger error (IP = %s): %v.", node.IP, err)
		return err
	}
	node.finger_lock.Lock()
	if node.finger[node.fix_index] != ip { //更新finger
		node.finger[node.fix_index] = ip
	}
	node.finger_lock.Unlock()
	node.fix_index = (node.fix_index-1)%159 + 2
	return nil
}

// 控制节点周期性地进行修复
func (node *Node) maintain() {
	go func() {
		for node.Online {
			node.block.Lock()
			node.stabilize()
			node.block.Unlock()
			time.Sleep(5 * time.Millisecond)
		}
		logrus.Infof("Node (IP = %s) stops stablizing.", node.IP)
	}()
	go func() {
		for node.Online {
			node.fix_finger()
			time.Sleep(5 * time.Millisecond)
		}
		logrus.Infof("Node (IP = %s) stops fixing finger.", node.IP)
	}()
}

// 更改前驱
func (node *Node) Change_predecessor(ip string) {
	node.pre_lock.Lock()
	node.predecessor = ip
	node.pre_lock.Unlock()
}

// 更改后继列表
func (node *Node) Change_successor_list(list [3]string) {
	node.suc_lock.Lock()
	for i := 0; i < 3; i++ { //将后继列表转移
		node.successor_list[i] = list[i]
	}
	node.suc_lock.Unlock()
}

// 得到后继列表
func (node *Node) Get_successor_list() [3]string {
	var successor_list [3]string
	node.suc_lock.RLock()
	for i := 0; i < 3; i++ {
		successor_list[i] = node.successor_list[i]
	}
	node.suc_lock.RUnlock()
	return successor_list
}

// 更新后继列表，使其中节点在线
func (node *Node) update_successor_list() error {
	var tmp, next_successor_list [3]string
	node.suc_lock.RLock()
	for i, ip := range node.successor_list {
		tmp[i] = ip
	}
	node.suc_lock.RUnlock()
	for _, ip := range tmp {
		if Ping(ip) { //找到最近的存在的后继
			err := Remote_call(ip, "DHT.Get_successor_list", Null{}, &next_successor_list)
			if err != nil {
				logrus.Errorf("Getting successor list error (IP = %s): %v.", node.IP, err)
				continue
			}
			node.suc_lock.Lock()
			node.successor_list[0] = ip //更新后继列表
			for j := 1; j < 3; j++ {
				node.successor_list[j] = next_successor_list[j-1]
			}
			node.suc_lock.Unlock()
			return nil
		}
	}
	logrus.Infof("All successors are offline (IP = %s).", node.IP)
	//后继列表中节点均失效，尝试通过路由表寻找以防止环断裂
	ip, err := node.Find_successor(cal(node.ID, 0))
	if err != nil {
		logrus.Errorf("Finding successor list error (IP = %s): %v.", node.IP, err)
	}
	node.suc_lock.Lock()
	for i := range node.successor_list {
		if i == 0 && ip != "" && ip != "OFFLINE" {
			node.successor_list[0] = ip
		} else {
			node.successor_list[i] = node.IP
		}
	}
	node.suc_lock.Unlock()
	return nil
}

// 上锁，以阻塞stabilize
func (node *Node) Lock() {
	node.block.Lock()
}

// 解锁，以恢复stabilize
func (node *Node) Unlock() {
	node.block.Unlock()
}

// 将数据存为某节点的主数据
func (node *Node) Put_in(data []Data_pair) error {
	node.data_lock.Lock()
	for i := range data {
		node.data[data[i].Key] = data[i].Value_pair
	}
	node.data_lock.Unlock()
	return nil
}

// 将数据存为某节点的备份数据
func (node *Node) Put_in_backup(data []Data_pair) error {
	node.data_backup_lock.Lock()
	for i := range data {
		node.data_backup[data[i].Key] = data[i].Value_pair
	}
	node.data_backup_lock.Unlock()
	return nil
}

// 将数据存为某节点的主数据，以及该节点的后继的备份数据
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

// 在某节点主数据中查询数据
func (node *Node) Get_out(key string) (string, bool) {
	node.data_lock.RLock()
	value_pair, ok := node.data[key]
	node.data_lock.RUnlock()
	return value_pair.Value, ok
}

// 在某节点中删除主数据，在其后继删除备份数据
func (node *Node) Delete_off_all(keys []string) bool {
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

// 在某节点中删除备份数据
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

// 得到hash值
func get_hash(Addr_IP string) *big.Int {
	hash := sha1.Sum([]byte(Addr_IP))
	hashInt := new(big.Int)
	return hashInt.SetBytes(hash[:])
}

// 判断是否在目标区间内
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

// 计算n+2^i并对2^160取模
func cal(n *big.Int, i int) *big.Int {
	tmp := new(big.Int).Add(n, exp[i])
	if tmp.Cmp(exp[160]) >= 0 {
		tmp.Sub(tmp, exp[160])
	}
	return tmp
}
