package chord

import (
	"math/big"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"dht/rpc"
)

type ValuePair struct {
	Value string
	KeyId *big.Int
}

type DataPair struct {
	Key    string
	Values ValuePair
}

type IpPair struct {
	IpPre    string
	IpPrePre string
}

type Null struct{}

type Node struct {
	Online         bool
	RPC            rpc.NodeRpc
	IP             string
	ID             *big.Int
	predecessor    string
	preLock        sync.RWMutex
	successorList  [3]string
	sucLock        sync.RWMutex
	finger         [161]string
	fingerLock     sync.RWMutex
	data           map[string]ValuePair
	dataLock       sync.RWMutex
	dataBackup     map[string]ValuePair
	dataBackupLock sync.RWMutex
	fixIndex       int
	fingerStart    [161]*big.Int
	block          sync.Mutex
	start          chan bool
	quit           chan bool
}

// 整体初始化
func init() {
	f, _ := os.Create("dht-test.log")
	logrus.SetOutput(f)
	initCal()
}

// 节点初始化
func (node *Node) Init(ip string) bool {
	node.Online = false
	node.start = make(chan bool, 1)
	node.quit = make(chan bool, 1)
	node.predecessor = ""
	node.IP = ip
	node.ID = getHash(ip)
	node.fixIndex = 2
	for i := 2; i <= 160; i++ {
		node.fingerStart[i] = cal(node.ID, i-1)
	}
	node.fingerLock.Lock()
	for i := range node.finger {
		node.finger[i] = node.IP
	}
	node.fingerLock.Unlock()
	node.sucLock.Lock()
	for i := range node.successorList {
		node.successorList[i] = node.IP
	}
	node.sucLock.Unlock()
	node.dataLock.Lock()
	node.data = make(map[string]ValuePair)
	node.dataLock.Unlock()
	node.dataBackupLock.Lock()
	node.dataBackup = make(map[string]ValuePair)
	node.dataBackupLock.Unlock()
	return true
}

// 节点开始运作
func (node *Node) Run() {
	node.Online = true
	go node.RPC.Serve(node.IP, "DHT", node.start, node.quit, &RPCWrapper{node})
}

// 创建DHT网络(加入第一个节点)
func (node *Node) Create() {
	rand.Seed(time.Now().UnixNano())
	logrus.Infof("Create a new DHT net.")
	node.preLock.Lock()
	node.predecessor = node.IP
	node.preLock.Unlock()
	select { // 阻塞直至run()完成
	case <-node.start:
		logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.IP, node.ID)
	}
	node.maintain()
}

// 向DHT网络中加入节点
func (node *Node) Join(ip string) bool {
	node.preLock.Lock()
	node.predecessor = ""
	node.preLock.Unlock()
	successor := ""
	select { // 阻塞直至run()完成
	case <-node.start:
		logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.IP, node.ID)
	}
	err := node.RPC.RemoteCall(ip, "DHT.FindSuccessor", node.ID, &successor)
	if err != nil {
		logrus.Errorf("Join error (IP = %s): %v.", node.IP, err)
		return false
	}
	node.sucLock.Lock()
	node.successorList[0] = successor
	node.sucLock.Unlock()
	node.maintain()
	return true
}

// 正常退出DHT网络(通知其他节点)
func (node *Node) Quit() {
	if !node.Online {
		return
	}
	node.block.Lock() //阻塞stablize
	node.preLock.RLock()
	predecessor := node.predecessor
	node.preLock.RUnlock()
	node.updateSuccessorList()
	var successorList [3]string
	node.sucLock.RLock()
	successorList[0] = node.successorList[0]
	successorList[1] = node.successorList[1]
	successorList[2] = node.successorList[2]
	node.sucLock.RUnlock()
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		if predecessor != "" && predecessor != "OFFLINE" && predecessor != node.IP {
			node.RPC.RemoteCall(predecessor, "DHT.Lock", Null{}, &Null{})
		}
		if predecessor != "" && predecessor != "OFFLINE" {
			node.RPC.RemoteCall(predecessor, "DHT.ChangeSuccessorList", successorList, &Null{})
		}
		wg.Done()
	}()
	go func() {
		if predecessor != successorList[0] && successorList[0] != node.IP {
			node.RPC.RemoteCall(successorList[0], "DHT.Lock", Null{}, &Null{})
		}
		node.RPC.RemoteCall(successorList[0], "DHT.ChangePredecessor", predecessor, &Null{})
		wg.Done()
	}()
	go func() { //转移主数据
		node.dataLock.RLock()
		data := []DataPair{}
		keys := []string{}
		for key, valuePair := range node.data {
			data = append(data, DataPair{key, valuePair})
			keys = append(keys, key)
		}
		node.dataLock.RUnlock()
		node.RPC.RemoteCall(successorList[0], "DHT.PutInAll", data, &Null{})
		node.RPC.RemoteCall(successorList[0], "DHT.DeleteOffBackup", keys, &Null{})
		wg.Done()
	}()
	go func() { //转移备份
		node.dataBackupLock.RLock()
		dataBackup := []DataPair{}
		for key, valuePair := range node.dataBackup {
			dataBackup = append(dataBackup, DataPair{key, valuePair})
		}
		node.dataBackupLock.RUnlock()
		node.RPC.RemoteCall(successorList[0], "DHT.PutInBackup", dataBackup, &Null{})
		wg.Done()
	}()
	wg.Wait()
	if predecessor != "" && predecessor != "OFFLINE" {
		node.RPC.RemoteCall(predecessor, "DHT.Unlock", Null{}, &Null{})
	}
	if predecessor != successorList[0] {
		node.RPC.RemoteCall(successorList[0], "DHT.Unlock", Null{}, &Null{})
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
	id := getHash(key)
	ip, _ := node.FindSuccessor(id)
	err := node.RPC.RemoteCall(ip, "DHT.PutInAll", []DataPair{{key, ValuePair{value, id}}}, &Null{})
	if err != nil {
		logrus.Errorf("Node (IP = %s) putting in data error (key = %s, value = %s): %v.", node.IP, key, value, err)
		return false
	}
	logrus.Infof("Node (IP = %s) puts in data : key = %s, value = %s.", node.IP, key, value)
	return true
}

// 查询数据
func (node *Node) Get(key string) (ok bool, value string) {
	id := getHash(key)
	node.preLock.RLock()
	pre := node.predecessor
	node.preLock.RUnlock()
	if belong(false, true, getHash(pre), node.ID, id) {
		value, ok = node.GetOut(key)
	} else {
		ip, _ := node.FindSuccessor(id)
		err := node.RPC.RemoteCall(ip, "DHT.GetOut", key, &value)
		if err != nil {
			logrus.Errorf("Node (IP = %s) getting out data error (key = %s, value = %s): %v.", node.IP, key, value, err)
			return false, ""
		}
		ok = true
	}
	logrus.Infof("Node (IP = %s) gets out data : key = %s, value = %s.", node.IP, key, value)
	return ok, value
}

// 删除数据
func (node *Node) Delete(key string) bool {
	id := getHash(key)
	ip, _ := node.FindSuccessor(id)
	err := node.RPC.RemoteCall(ip, "DHT.DeleteOffAll", []string{key}, &Null{})
	if err != nil {
		logrus.Errorf("Node (IP = %s) deleting off data error (key = %s): %v.", node.IP, key, err)
		return false
	}
	logrus.Infof("Node (IP = %s) delete off data : key = %s.", node.IP, key)
	return true
}

// 找某个地址id的后继节点
func (node *Node) FindSuccessor(id *big.Int) (ip string, err error) {
	if id.Cmp(node.ID) == 0 { //当前节点即位目标后继，结束
		ip = node.IP
		return ip, nil
	}
	pre, err := node.FindPredecessor(id)
	if err != nil {
		logrus.Errorf("FindSuccessor error (IP = %s): %v.", node.IP, err)
		return "", err
	}
	err = node.RPC.RemoteCall(pre, "DHT.GetSuccessor", Null{}, &ip)
	if err != nil {
		logrus.Errorf("FindSuccessor error (IP = %s): %v.", node.IP, err)
		return "", err
	}
	return ip, nil
}

// 找某个地址id的前驱节点
func (node *Node) FindPredecessor(id *big.Int) (ip string, err error) {
	ip = node.IP
	node.sucLock.RLock()
	successorId := getHash(node.successorList[0])
	node.sucLock.RUnlock()
	if !belong(false, true, node.ID, successorId, id) {
		err = node.RPC.RemoteCall(node.closestPrecedingFinger(id), "DHT.FindPredecessor", id, &ip)
		if err != nil {
			logrus.Errorf("FindPredecessor error (IP = %s): %v.", node.IP, err)
			return "", err
		}
	}
	return ip, nil
}

// 得到某节点的后继(同时顺带更新successorList)
func (node *Node) GetSuccessor() (ip string, err error) {
	node.updateSuccessorList()
	node.sucLock.Lock()
	defer node.sucLock.Unlock()
	return node.successorList[0], nil
}

// 得到某节点的前驱(同时顺带监测前驱是否异常下线，若发现则利用节点的备份数据进行数据恢复)
func (node *Node) GetPredecessor() (string, error) {
	node.preLock.RLock()
	predecessor := node.predecessor
	node.preLock.RUnlock()
	if predecessor != "" && !node.Ping(predecessor) {
		node.preLock.RLock()
		node.preLock.RUnlock()
		node.preLock.Lock()
		node.predecessor = "OFFLINE" //若前驱已下线，置上标记
		predecessor = "OFFLINE"
		node.preLock.Unlock()
		dataBackup := []DataPair{}
		node.dataBackupLock.Lock()
		for key, valuePair := range node.dataBackup {
			dataBackup = append(dataBackup, DataPair{key, valuePair})
		}
		for _, dataPair := range dataBackup {
			delete(node.dataBackup, dataPair.Key) //从备份中删去
		}
		node.dataBackupLock.Unlock()
		node.dataLock.Lock()
		for _, dataPair := range dataBackup {
			node.data[dataPair.Key] = dataPair.Values //加入主数据
		}
		node.dataLock.Unlock()
		successor, err := node.GetSuccessor()
		if err != nil {
			logrus.Errorf("Restoring data error (IP = %s): %v.", node.IP, err)
			return "OFFLINE", err
		}
		err = node.RPC.RemoteCall(successor, "DHT.PutInBackup", dataBackup, &Null{})
		if err != nil {
			logrus.Errorf("Restoring data error (IP = %s): %v.", node.IP, err)
			return "OFFLINE", err
		}
	}
	return predecessor, nil
}

// 找到路由表中在目标位置前最近的上线节点
func (node *Node) closestPrecedingFinger(id *big.Int) string {
	for i := 160; i > 1; i-- {
		if !node.Ping(node.finger[i]) { //已下线，将finger设为仍然上线的位置
			if i == 160 {
				node.finger[i] = node.IP
			} else {
				node.finger[i] = node.finger[i+1]
			}
		} else if belong(false, false, node.ID, id, getHash(node.finger[i])) {
			return node.finger[i]
		}
	}
	node.sucLock.Lock()
	defer node.sucLock.Unlock()
	if !node.Ping(node.successorList[0]) { //已下线，将finger设为仍然上线的位置
		node.successorList[0] = node.IP
	} else if belong(false, false, node.ID, id, getHash(node.successorList[0])) {
		return node.successorList[0]
	}
	return node.IP
}

// 测试节点是否上线
func (node *Node) Ping(ip string) bool {
	err := node.RPC.RemoteCall(ip, "DHT.Ping", Null{}, &Null{})
	return err == nil
}

// 修护前驱后继，并相应地转移数据
func (node *Node) stabilize() error {
	successor, err := node.GetSuccessor()
	if err != nil {
		logrus.Errorf("Stabilize error (IP = %s): %v.", node.IP, err)
		return err
	}
	ip := ""
	err = node.RPC.RemoteCall(successor, "DHT.GetPredecessor", Null{}, &ip)
	if err != nil {
		logrus.Errorf("Stabilize error (IP = %s): %v.", node.IP, err)
		return err
	}
	if (successor == node.IP) ||
		(ip != "" && ip != "OFFLINE" && belong(false, false, node.ID, getHash(successor), getHash(ip))) {
		node.sucLock.Lock() //需要更改后继
		node.successorList[2] = node.successorList[1]
		node.successorList[1] = node.successorList[0]
		node.successorList[0] = ip
		node.sucLock.Unlock()
		node.RPC.RemoteCall(successor, "DHT.TransferData", IpPair{ip, node.IP}, &Null{})
	}
	node.sucLock.RLock()
	successor = node.successorList[0]
	node.sucLock.RUnlock()
	err = node.RPC.RemoteCall(successor, "DHT.Notifty", node.IP, &Null{})
	if err != nil {
		logrus.Errorf("Stabilize error (IP = %s): %v.", node.IP, err)
		return err
	}
	if ip == "OFFLINE" {
		data := []DataPair{}
		node.dataLock.RLock()
		for key, valuePair := range node.data {
			data = append(data, DataPair{key, valuePair})
		}
		node.dataLock.RUnlock()
		err = node.RPC.RemoteCall(successor, "DHT.PutInBackup", data, &Null{})
		if err != nil {
			logrus.Errorf("Restoring data error (IP = %s): %v.", node.IP, err)
			return err
		}
	}
	return nil
}

// 修复前驱
func (node *Node) Notifty(ip string) error {
	node.preLock.Lock()
	if node.predecessor == "" || node.predecessor == "OFFLINE" ||
		belong(false, false, getHash(node.predecessor), node.ID, getHash(ip)) {
		node.predecessor = ip
	}
	node.preLock.Unlock()
	return nil
}

// 更改后继时，转移数据
func (node *Node) TransferData(ips IpPair) error { //将数据转移到前节点
	preId := getHash(ips.IpPre)
	prePreId := getHash(ips.IpPrePre)
	//转移主数据
	data := []DataPair{}
	node.dataLock.RLock()
	for key, valuePair := range node.data {
		if !belong(false, true, preId, node.ID, valuePair.KeyId) {
			data = append(data, DataPair{key, valuePair})
		}
	}
	node.dataLock.RUnlock()
	node.dataLock.Lock()
	for _, dataPair := range data {
		delete(node.data, dataPair.Key)
	}
	node.dataLock.Unlock()
	err := node.RPC.RemoteCall(ips.IpPre, "DHT.PutIn", data, &Null{})
	if err != nil {
		logrus.Errorf("Node (IP = %s , toIp = %s) transferring data error: %v.", node.IP, ips.IpPre, err)
		return err
	}
	//转移主数据对应的备份
	keyBackup := []string{}
	node.dataBackupLock.Lock()
	for _, dataPair := range data {
		node.dataBackup[dataPair.Key] = dataPair.Values
		keyBackup = append(keyBackup, dataPair.Key)
	}
	node.dataBackupLock.Unlock()
	successor, _ := node.GetSuccessor()
	err = node.RPC.RemoteCall(successor, "DHT.DeleteOffBackup", keyBackup, &Null{})
	if err != nil {
		logrus.Errorf("Node (IP = %s , sucIp = %s) transferring backup data error: %v.", node.IP, successor, err)
		return err
	}
	//转移备份
	dataBackup := []DataPair{}
	node.dataBackupLock.RLock()
	for key, valuePair := range node.dataBackup {
		if !belong(false, true, prePreId, preId, valuePair.KeyId) {
			dataBackup = append(dataBackup, DataPair{key, valuePair})
		}
	}
	node.dataBackupLock.RUnlock()
	node.dataBackupLock.Lock()
	for _, dataPair := range dataBackup {
		delete(node.dataBackup, dataPair.Key)
	}
	node.dataBackupLock.Unlock()
	err = node.RPC.RemoteCall(ips.IpPre, "DHT.PutInBackup", dataBackup, &Null{})
	if err != nil {
		logrus.Errorf("Node (IP = %s , toIp = %s) transferring backup data error: %v.", node.IP, ips.IpPre, err)
		return err
	}
	return nil
}

// 修复路由表
func (node *Node) fixFinger() error {
	ip, err := node.FindSuccessor(node.fingerStart[node.fixIndex])
	if err != nil {
		logrus.Errorf("Fixing finger error (IP = %s): %v.", node.IP, err)
		return err
	}
	node.fingerLock.Lock()
	if node.finger[node.fixIndex] != ip { //更新finger
		node.finger[node.fixIndex] = ip
	}
	node.fingerLock.Unlock()
	node.fixIndex = (node.fixIndex-1)%159 + 2
	return nil
}

// 控制节点周期性地进行修复
func (node *Node) maintain() {
	go func() {
		for node.Online {
			node.block.Lock()
			node.stabilize()
			node.block.Unlock()
			time.Sleep(50 * time.Millisecond)
		}
		logrus.Infof("Node (IP = %s) stops stablizing.", node.IP)
	}()
	go func() {
		for node.Online {
			node.block.Lock()
			node.fixFinger()
			node.block.Unlock()
			time.Sleep(50 * time.Millisecond)
		}
		logrus.Infof("Node (IP = %s) stops fixing finger.", node.IP)
	}()
}

// 更改前驱
func (node *Node) ChangePredecessor(ip string) {
	node.preLock.Lock()
	node.predecessor = ip
	node.preLock.Unlock()
}

// 更改后继列表
func (node *Node) ChangeSuccessorList(list [3]string) {
	node.sucLock.Lock()
	for i := 0; i < 3; i++ { //将后继列表转移
		node.successorList[i] = list[i]
	}
	node.sucLock.Unlock()
}

// 得到后继列表
func (node *Node) GetSuccessorList() [3]string {
	var successorList [3]string
	node.sucLock.RLock()
	for i := 0; i < 3; i++ {
		successorList[i] = node.successorList[i]
	}
	node.sucLock.RUnlock()
	return successorList
}

// 更新后继列表，使其中节点在线
func (node *Node) updateSuccessorList() error {
	var tmp, nextSuccessorList [3]string
	node.sucLock.RLock()
	for i, ip := range node.successorList {
		tmp[i] = ip
	}
	node.sucLock.RUnlock()
	for _, ip := range tmp {
		if node.Ping(ip) { //找到最近的存在的后继
			err := node.RPC.RemoteCall(ip, "DHT.GetSuccessorList", Null{}, &nextSuccessorList)
			if err != nil {
				logrus.Errorf("Getting successor list error (IP = %s): %v.", node.IP, err)
				continue
			}
			node.sucLock.Lock()
			node.successorList[0] = ip //更新后继列表
			for j := 1; j < 3; j++ {
				node.successorList[j] = nextSuccessorList[j-1]
			}
			node.sucLock.Unlock()
			return nil
		}
	}
	logrus.Infof("All successors are offline (IP = %s).", node.IP)
	//后继列表中节点均失效，尝试通过路由表寻找以防止环断裂
	ip, err := node.FindSuccessor(cal(node.ID, 0))
	if err != nil {
		logrus.Errorf("Finding successor list error (IP = %s): %v.", node.IP, err)
	}
	node.sucLock.Lock()
	for i := range node.successorList {
		if i == 0 && ip != "" && ip != "OFFLINE" {
			node.successorList[0] = ip
		} else {
			node.successorList[i] = node.IP
		}
	}
	node.sucLock.Unlock()
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
func (node *Node) PutIn(data []DataPair) error {
	node.dataLock.Lock()
	for i := range data {
		node.data[data[i].Key] = data[i].Values
	}
	node.dataLock.Unlock()
	return nil
}

// 将数据存为某节点的备份数据
func (node *Node) PutInBackup(data []DataPair) error {
	node.dataBackupLock.Lock()
	for i := range data {
		node.dataBackup[data[i].Key] = data[i].Values
	}
	node.dataBackupLock.Unlock()
	return nil
}

// 将数据存为某节点的主数据，以及该节点的后继的备份数据
func (node *Node) PutInAll(data []DataPair) error {
	node.PutIn(data)
	//后继节点放入备份数据
	successor, err := node.GetSuccessor()
	if err != nil {
		logrus.Errorf("Getting successor error (IP = %s): %v.", node.IP, err)
		return err
	}
	err = node.RPC.RemoteCall(successor, "DHT.PutInBackup", data, &Null{})
	if err != nil {
		logrus.Errorf("Putting in backup error (IP = %s): %v.", node.IP, err)
	}
	return nil
}

// 在某节点主数据中查询数据
func (node *Node) GetOut(key string) (string, bool) {
	node.dataLock.RLock()
	valuePair, ok := node.data[key]
	node.dataLock.RUnlock()
	return valuePair.Value, ok
}

// 在某节点中删除主数据，在其后继删除备份数据
func (node *Node) DeleteOffAll(keys []string) bool {
	out := true
	node.dataLock.Lock()
	for _, key := range keys {
		_, ok := node.data[key]
		if !ok {
			out = false
		} else {
			delete(node.data, key)
		}
	}
	node.dataLock.Unlock()
	//删除后继节点的备份数据
	successor, err := node.GetSuccessor()
	if err != nil {
		logrus.Errorf("Getting successor error (IP = %s): %v.", node.IP, err)
		return false
	}
	err = node.RPC.RemoteCall(successor, "DHT.DeleteOffBackup", keys, &Null{})
	if err != nil {
		logrus.Errorf("Deleting off backup error (IP = %s): %v.", node.IP, err)
		return false
	}
	return out
}

// 在某节点中删除备份数据
func (node *Node) DeleteOffBackup(keys []string) bool {
	out := true
	node.dataBackupLock.Lock()
	for _, key := range keys {
		_, ok := node.dataBackup[key]
		if !ok {
			out = false
		} else {
			delete(node.dataBackup, key)
		}
	}
	node.dataBackupLock.Unlock()
	return out
}
