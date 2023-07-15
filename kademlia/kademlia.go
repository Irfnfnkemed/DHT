package kademlia

import (
	"dht/rpc"
	"math/big"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

type callArgs struct {
	ip   string
	args interface{}
}

type ValuePair struct {
	Value string
	KeyId *big.Int
}

type DataPair struct {
	Key    string
	Values ValuePair
}

type Node struct {
	Online   bool
	RPC      rpc.NodeRpc
	IP       string
	ID       *big.Int
	buckets  [160]Bucket
	data     map[string]ValuePair
	dataLock sync.RWMutex
	start    chan bool
	quit     chan bool
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
	node.data = make(map[string]ValuePair)
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
func (node *Node) findNode(ip string) []string {
	i := belong(node.ID, getHash(ip))
	nodeList := []string{}
	for p := node.buckets[i].begin(); p != node.buckets[i].end(); p = p.next {
		nodeList = append(nodeList, p.ip)
	}
	if len(nodeList) == k {
		return nodeList
	}
	for j := i + 1; j < 160; j++ {
		for p := node.buckets[j].begin(); p != node.buckets[j].end(); p = p.next {
			nodeList = append(nodeList, p.ip)
			if len(nodeList) == k {
				return nodeList
			}
		}
	}
	for j := i - 1; j >= 0; j-- {
		for p := node.buckets[j].begin(); p != node.buckets[j].end(); p = p.next {
			nodeList = append(nodeList, p.ip)
			if len(nodeList) == k {
				return nodeList
			}
		}
	}
	return nodeList
}

// 测试节点是否上线
func Ping(ipFrom, ipTo string) bool {
	err := rpc.RemoteCall(ipTo, "DHT.Ping", ipFrom, &Null{})
	return err == nil
}

func (node *Node) flush(ip string) {
	i := belong(node.ID, getHash(ip))
	node.buckets[i].flush(ip)
}
