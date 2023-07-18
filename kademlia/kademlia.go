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

type IpPairs struct {
	IpFrom string
	IpTo   string
}

type IpDataPairs struct {
	IpFrom string
	Datas  DataPair
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

func (node *Node) Init(ip string) error {
	node.Online = false
	node.IP = ip
	node.ID = getHash(ip)
	for i := range node.buckets {
		node.buckets[i].init(node.IP)
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

// 找到节点已知的距离目标最近的k个节点ip
func (node *Node) FindNode(ip string) []string {
	i := belong(node.ID, getHash(ip))
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
	for j := i - 1; j >= 0; j-- {
		tmpList := node.buckets[j].getAll()
		for _, ipNode := range tmpList {
			nodeList = append(nodeList, ipNode)
			if len(nodeList) == k {
				return nodeList
			}
		}
	}
	for j := i + 1; j < 160; j++ {
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

// 测试节点是否上线
func Ping(ipTo string) bool {
	err := rpc.RemoteCall(ipTo, "DHT.Ping", Null{}, &Null{})
	return err == nil
}

func (node *Node) flush(ip string, online bool) {
	i := belong(node.ID, getHash(ip))
	if i != -1 {
		node.buckets[i].flush(ip, online)
	}
}

// 找到系统中距离目标最近的k个节点ip
func (node *Node) nodeLookup(ip string) []string {
	order := Order{}
	order.init(ip)
	list := node.FindNode(ip)
	for _, ipFind := range list {
		order.insert(ipFind)
	}
	for {
		callList := order.get()
		findList := node.findNodeList(&order, callList, ip)
		flag := order.flush(findList) //更新order
		if !flag {
			callList = order.getUndone()
			findList = node.findNodeList(&order, callList, ip)
			flag = order.flush(findList) //更新order
		}
		if !flag {
			break
		}
	}
	return order.getClosest()
}

func (node *Node) findNodeList(order *Order, callList []*orderUnit, ipTarget string) []string {
	findList := []string{}
	for _, p := range callList {
		p.done = true
		err := rpc.RemoteCall(p.ip, "DHT.FindNode", IpPairs{node.IP, ipTarget}, &findList)
		node.flush(p.ip, err == nil)
		if err != nil {
			logrus.Errorf("FindNode error, server IP = %s", p.ip)
			order.delete(p)
		}
	}
	return findList
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

func (node *Node) Join(ip string) bool {
	select { // 阻塞直至run()完成
	case <-node.start:
		logrus.Infof("New node (IP = %s, ID = %v) joins in.", node.IP, node.ID)
	}
	i := belong(node.ID, getHash(ip))
	node.buckets[i].insertToHead(ip)
	node.nodeLookup(node.IP) //通过查找自身，更新路由表
	node.maintain()
	return true
}

func (node *Node) Put(key string, value string) bool {
	nodeList := node.nodeLookup(key)
	flag := false
	for _, ip := range nodeList {
		if node.IP == ip {
			node.data.put(key, value)
			flag = true
		} else {
			err := rpc.RemoteCall(ip, "DHT.PutIn", IpDataPairs{node.IP, DataPair{key, value}}, &Null{})
			if err != nil {
				logrus.Errorf("Putting in error, IP = %s: %v", ip, err)
			}
			node.flush(ip, err == nil)
			if err == nil {
				flag = true
			}
		}
	}
	return flag
}

func (node *Node) PutIn(dataPair DataPair) {
	node.data.put(dataPair.Key, dataPair.Value)
}

func (node *Node) republish() {
	republishList := node.data.getRepublishList()
	var wg sync.WaitGroup
	wg.Add(len(republishList))
	for _, dataPair := range republishList {
		go node.republishData(dataPair, &wg)
	}
	wg.Wait()
	time.Sleep(500 * time.Millisecond)
	logrus.Infof("Node (IP = %s) republishes the data.", node.IP)
}

func (node *Node) republishData(dataPair DataPair, wg *sync.WaitGroup) {
	nodeList := node.nodeLookup(dataPair.Key)
	for _, ip := range nodeList {
		if ip == node.IP {
			node.FlushData(dataPair)
		} else {
			err := rpc.RemoteCall(ip, "DHT.FlushData", IpDataPairs{node.IP, dataPair}, &Null{})
			if err != nil {
				logrus.Errorf("Republishing error, IP = %s: %v", ip, err)
			}
			node.flush(ip, err == nil)
		}
	}
	wg.Done()
}

func (node *Node) FlushData(dataPair DataPair) {
	node.data.flush(dataPair)
}

func (node *Node) abandon() {
	node.data.abandon()
}

func (node *Node) maintain() {
	go func() {
		for node.Online {
			node.republish()
			time.Sleep(10 * time.Second)
		}
		logrus.Infof("Node (IP = %s) stops republishing.", node.IP)
	}()
	go func() {
		for node.Online {
			node.abandon()
			time.Sleep(10 * time.Second)
		}
		logrus.Infof("Node (IP = %s) stops abandoning.", node.IP)
	}()
}

func (node *Node) Getout(key string) (bool, string) {
	value, ok := node.data.get(key)
	return ok, value
}

func (node *Node) Get(key string) (bool, string) {
	order := Order{}
	order.init(key)
	list := node.FindNode(key)
	for _, ipFind := range list {
		order.insert(ipFind)
	}
	for {
		callList := order.get()
		findList, value := node.findValueList(&order, callList, key)
		if value != "" {
			return true, value
		}
		flag := order.flush(findList) //更新order
		if !flag {
			callList = order.getUndone()
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

func (node *Node) findValueList(order *Order, callList []*orderUnit, key string) (findList []string, value string) {
	for _, p := range callList {
		p.done = true
		err := rpc.RemoteCall(p.ip, "DHT.FindNode", IpPairs{node.IP, key}, &findList)
		node.flush(p.ip, err == nil)
		if err != nil {
			logrus.Errorf("FindNode error, server IP = %s", p.ip)
			order.delete(p)
			continue
		}
		err = rpc.RemoteCall(p.ip, "DHT.Getout", IpPairs{node.IP, key}, &value)
		if err != nil {
			logrus.Errorf("findValueList error, server IP = %s", p.ip)
			continue
		}
		if value != "" {
			return []string{}, value
		}
	}
	return findList, ""
}

func (node *Node) Delete(key string) bool {
	///////////////////////////////////////
	return true
}

func (node *Node) ForceQuit() {
	if !node.Online {
		return
	}
	node.Online = false
	close(node.quit)
	logrus.Infof("Node (IP = %s, ID = %v) force quits.", node.IP, node.ID)
}

func (node *Node) Quit() {
	if !node.Online {
		return
	}
	node.Online = false
	close(node.quit)
	node.republish()
	logrus.Infof("Node (IP = %s, ID = %v) quits.", node.IP, node.ID)
}

// func (node *Node) refresh() {
// 	if node.buckets[node.refreshIndex].getSize() <= 1 {
//         node.nodeLookup()
// 	}
// }
