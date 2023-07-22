package kademlia

import (
	"dht/rpc"
	"math/big"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const k = 16 //bucket的大小
const a = 3  //NodeLookup中的alpha大小
const RepulishCircleTime = 4 * time.Second
const AbandonCircleTime = 15 * time.Second
const RefreshCircleTime = 250 * time.Millisecond

type Null struct{}

type IpPairs struct {
	IpFrom string
	IpTo   string
}

type IpDataPairs struct {
	IpFrom string
	Datas  DataPair
}

type IpIdPairs struct {
	IpFrom string
	IdTo   *big.Int
}

type Node struct {
	Online       bool
	RPC          rpc.NodeRpc
	IP           string
	ID           *big.Int
	buckets      [160]Bucket
	refreshIndex int
	data         Data
	start        chan bool
	quit         chan bool
}

// 整体初始化
func init() {
	f, _ := os.Create("dht-test.log")
	logrus.SetOutput(f)
	initCal()
}

// 节点初始化
func (node *Node) Init(ip string) error {
	node.Online = false
	node.IP = ip
	node.ID = getHash(ip)
	for i := range node.buckets {
		node.buckets[i].init(node.IP, node)
	}
	node.refreshIndex = 0
	node.data.Init()
	node.start = make(chan bool, 1)
	node.quit = make(chan bool, 1)
	return nil
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
	select { // 阻塞直至run()完成
	case <-node.start:
		logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.IP, node.ID)
	}
	node.maintain()
}

// 加入节点
func (node *Node) Join(ip string) bool {
	select { // 阻塞直至run()完成
	case <-node.start:
		logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.IP, node.ID)
	}
	i := belong(node.ID, getHash(ip))
	node.buckets[i].insertToHead(ip)
	node.nodeLookup(node.ID) //通过查找自身，更新路由表
	node.maintain()
	return true
}

// 存入数据
func (node *Node) Put(key string, value string) bool {
	nodeList := node.nodeLookup(getHash(key))
	flag := false
	var wg sync.WaitGroup
	wg.Add(len(nodeList))
	for _, ip := range nodeList {
		go func(ip string) {
			defer wg.Done()
			if node.IP == ip {
				node.data.put(key, value)
				flag = true
			} else {
				err := node.RPC.RemoteCall(ip, "DHT.PutIn", IpDataPairs{node.IP, DataPair{key, value}}, &Null{})
				if err != nil {
					logrus.Errorf("Putting in error, IP = %s: %v", ip, err)
				}
				node.flush(ip, err == nil)
				if err == nil {
					flag = true
				}
			}
		}(ip)
	}
	wg.Wait()
	return flag
}

// 查找数据
func (node *Node) Get(key string) (bool, string) {
	order := Order{}
	order.init(getHash(key))
	list := node.FindNode(getHash(key))
	for _, ipFind := range list {
		order.insert(ipFind)
	}
	for {
		callList := order.getUndoneAlpha()
		findList, value := node.findValueList(&order, callList, key)
		if value != "" {
			return true, value
		}
		flag := order.flush(findList) //更新order
		if !flag {
			callList = order.getUndoneAll()
			findList, value = node.findValueList(&order, callList, key)
			if value != "" {
				return true, value
			}
			flag = order.flush(findList) //更新order
		}
		if !flag {
			break
		}
	}
	return false, ""
}

// 删除数据
func (node *Node) Delete(key string) bool {
	return true //由于Kademlia协议本身并不支持删除，因此删除操作是无效的
}

// 正常退出(可通知外界)
func (node *Node) Quit() {
	if !node.Online {
		return
	}
	node.Online = false
	close(node.quit)
	node.republish()
	logrus.Infof("Node (IP = %s, ID = %v) quits.", node.IP, node.ID)
}

// 异常退出(不通知外界)
func (node *Node) ForceQuit() {
	if !node.Online {
		return
	}
	node.Online = false
	close(node.quit)
	logrus.Infof("Node (IP = %s, ID = %v) force quits.", node.IP, node.ID)
}

// 测试节点是否上线
func (node *Node) Ping(ipTo string) bool {
	if ipTo == "" {
		return false
	}
	err := node.RPC.RemoteCall(ipTo, "DHT.Ping", Null{}, &Null{})
	return err == nil
}

// 刷新节点的bucket(用于remote call之后)
func (node *Node) flush(ip string, online bool) {
	i := belong(node.ID, getHash(ip))
	if i != -1 {
		node.buckets[i].flush(ip, online) //online表示目标节点是否上线
	}
}

// 找到节点已知的距离目标最近的k个节点ip
func (node *Node) FindNode(id *big.Int) []string {
	i := belong(node.ID, id)
	nodeList := []string{}
	if i == -1 {
		nodeList = append(nodeList, node.IP) //自身是最近的
	} else {
		tmpList := node.buckets[i].getAll()
		for _, ipNode := range tmpList {
			nodeList = append(nodeList, ipNode)
		}
	}
	if len(nodeList) == k {
		return nodeList
	}
	for j := i - 1; j >= 0; j-- { //从较小的桶里补充
		tmpList := node.buckets[j].getAll()
		for _, ipNode := range tmpList {
			nodeList = append(nodeList, ipNode)
			if len(nodeList) == k {
				return nodeList
			}
		}
	}
	for j := i + 1; j < 160; j++ { //从较大的桶里补充
		tmpList := node.buckets[j].getAll()
		for _, ipNode := range tmpList {
			nodeList = append(nodeList, ipNode)
			if len(nodeList) == k {
				return nodeList
			}
		}
	}
	if i != -1 {
		nodeList = append(nodeList, node.IP)
	}
	return nodeList
}

// 找到系统中距离目标最近的k个节点ip
func (node *Node) nodeLookup(id *big.Int) []string {
	order := Order{}
	order.init(id)
	list := node.FindNode(id)
	for _, ipFind := range list {
		order.insert(ipFind)
	}
	for {
		callList := order.getUndoneAlpha()
		findList := node.findNodeList(&order, callList, id)
		flag := order.flush(findList) //更新order
		if !flag {
			callList = order.getUndoneAll()
			findList = node.findNodeList(&order, callList, id)
			flag = order.flush(findList) //更新order
		}
		if !flag {
			break
		}
	}
	return order.getClosest()
}

// 给出可能的最近k个的目标的候补列表（用于NodeLookup）
func (node *Node) findNodeList(order *Order, callList []*orderUnit, idTarget *big.Int) []string {
	findList := []string{}
	var lock sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(callList))
	for _, p := range callList {
		go func(q *orderUnit) {
			subFindList := []string{}
			defer wg.Done()
			q.done = true
			err := node.RPC.RemoteCall(q.ip, "DHT.FindNode", IpIdPairs{node.IP, idTarget}, &subFindList)
			node.flush(q.ip, err == nil)
			if err != nil {
				logrus.Errorf("FindNode error, server IP = %s", q.ip)
				order.delete(q)
				return
			}
			lock.Lock()
			findList = append(findList, subFindList...)
			lock.Unlock()
		}(p)
	}
	wg.Wait()
	return findList
}

// 重新发布一个节点的数据
func (node *Node) republish() {
	republishList := node.data.getRepublishList()
	var wg sync.WaitGroup
	wg.Add(len(republishList))
	for _, dataPair := range republishList {
		go node.republishData(dataPair, &wg)
	}
	wg.Wait()
	time.Sleep(750 * time.Millisecond)
	logrus.Infof("Node (IP = %s) republishes the data.", node.IP)
}

// 发布一条数据
func (node *Node) republishData(dataPair DataPair, wg *sync.WaitGroup) {
	nodeList := node.nodeLookup(getHash(dataPair.Key))
	var wgPut sync.WaitGroup
	wgPut.Add(len(nodeList))
	for _, ip := range nodeList {
		go func(ip string) {
			defer wgPut.Done()
			if ip == node.IP {
				node.PutIn(dataPair)
			} else {
				err := node.RPC.RemoteCall(ip, "DHT.PutIn", IpDataPairs{node.IP, dataPair}, &Null{})
				if err != nil {
					logrus.Errorf("Republishing error, IP = %s: %v", ip, err)
				}
				node.flush(ip, err == nil)
			}
		}(ip)
	}
	wgPut.Wait()
	wg.Done()
}

// 将某条数据存入该节点
func (node *Node) PutIn(dataPair DataPair) {
	node.data.put(dataPair.Key, dataPair.Value)
}

// 查找某个节点的某条数据
func (node *Node) Getout(key string) (bool, string) {
	value, ok := node.data.get(key)
	return ok, value
}

// 给出可能的最近k个的目标的候补列表，若找到数据值，直接结束（用于Get）
func (node *Node) findValueList(order *Order, callList []*orderUnit, key string) (findList []string, value string) {
	for _, p := range callList {
		subFindList := []string{}
		p.done = true
		err := node.RPC.RemoteCall(p.ip, "DHT.FindNode", IpIdPairs{node.IP, getHash(key)}, &subFindList)
		node.flush(p.ip, err == nil)
		if err != nil {
			logrus.Errorf("FindNode error, server IP = %s", p.ip)
			order.delete(p)
			continue
		}
		err = node.RPC.RemoteCall(p.ip, "DHT.Getout", IpPairs{node.IP, key}, &value)
		if err != nil {
			logrus.Errorf("findValueList error, server IP = %s", p.ip)
			continue
		}
		if value != "" {
			return []string{}, value
		}
		findList = append(findList, subFindList...)
	}
	return findList, ""
}

// 舍弃过期数据
func (node *Node) abandon() {
	node.data.abandon()
}

// 定期检查bucket，尽量使得bucket不要为空
func (node *Node) refresh() {
	node.buckets[node.refreshIndex].check()
	if node.buckets[node.refreshIndex].getSize() < 2 {
		node.nodeLookup(exp[node.refreshIndex])
	}
	node.refreshIndex = (node.refreshIndex + 1) % 160
}

// 定期维护结构与数据分布
func (node *Node) maintain() {
	go func() {
		for node.Online {
			node.republish()
			time.Sleep(RepulishCircleTime)
		}
		logrus.Infof("Node (IP = %s) stops republishing.", node.IP)
	}()
	go func() {
		for node.Online {
			node.abandon()
			time.Sleep(AbandonCircleTime)
		}
		logrus.Infof("Node (IP = %s) stops abandoning.", node.IP)
	}()
	go func() {
		for node.Online {
			node.refresh()
			time.Sleep(RefreshCircleTime)
		}
		logrus.Infof("Node (IP = %s) stops refreshing.", node.IP)
	}()
}
