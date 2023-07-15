package rpc

import (
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var clientPool map[string]chan *rpc.Client //ip到用户池的映射
var clientPoolLock sync.RWMutex

type NodeRpc struct {
	server   *rpc.Server
	listener net.Listener
	clients  chan *rpc.Client //容量为20，容纳可用客户端
	conns    chan net.Conn    //容量为40，容纳Dial()、Accept()产生的连接，在停止服务时关闭
}

type Null struct{}

// 远端调用
func RemoteCall(ip string, serviceMethod string, args interface{}, reply interface{}) error {
	if serviceMethod != "DHT.Ping" {
		logrus.Infof("Remote call (server IP = %s, serviceMethod = %s).", ip, serviceMethod)
	}
	client, err := getClient(ip)
	defer returnClient(ip, client)
	if err != nil {
		if serviceMethod != "DHT.Ping" {
			logrus.Errorf("Getting client error (server IP = %s): %v.", ip, err)
		}
		return err
	}
	err = client.Call(serviceMethod, args, reply)
	if err != nil {
		if serviceMethod != "DHT.Ping" {
			logrus.Errorf("Calling error (server IP = %s): %v.", ip, err)
		}
		return err
	} else {
		if serviceMethod != "DHT.Ping" {
			logrus.Infof("Call done (server IP = %s, serviceMethod = %s).", ip, serviceMethod)
		}
	}
	return nil
}

// 节点服务
func (nodeRpc *NodeRpc) Serve(ip, serveName string, start, quit chan bool, registerNode interface{}) error {
	var err error = nil
	nodeRpc.server = rpc.NewServer()
	err = nodeRpc.server.RegisterName(serveName, registerNode) // registerNode是需要注册的节点的指针
	if err != nil {
		logrus.Errorf("Registing error (server IP = %s): %v.", ip, err)
		return err
	} else {
		logrus.Infof("Regist done (server IP = %s, name = %s).", ip, serveName)
	}
	nodeRpc.listener, err = net.Listen("tcp", ip)
	if err != nil {
		logrus.Errorf("Listening error (server IP = %s): %v.", ip, err)
		return err
	} else {
		logrus.Infof("Listen done (server IP = %s, network = tcp).", ip)
	}
	err = nodeRpc.createClient(ip)
	if err != nil {
		logrus.Errorf("Creating clients error (server IP = %s): %v.", ip, err)
		return err
	} else {
		logrus.Infof("Create clients done (server IP = %s, network = tcp).", ip)
	}
	close(start) //疏通开始通道
	select {
	case <-quit:
		nodeRpc.closeConn() //结束服务
	}
	logrus.Infof("Node stops serving (server IP = %s).", ip)
	nodeRpc.closeConn()
	nodeRpc.listener.Close()
	return nil
}

// 创建一个节点的用户池
func (nodeRpc *NodeRpc) createClient(ip string) error {
	nodeRpc.clients = make(chan *rpc.Client, 20)
	nodeRpc.conns = make(chan net.Conn, 40)
	for i := 0; i < 20; i++ {
		conn, err := net.DialTimeout("tcp", ip, time.Second)
		if err != nil {
			logrus.Errorf("Dialing error (server IP = %s): %v.", ip, err)
			continue
		}
		client := rpc.NewClient(conn)
		if nodeRpc.connect(client) != nil {
			logrus.Errorf("Connecting error (server IP = %s): %v.", ip, err)
			continue
		}
		nodeRpc.conns <- conn
	}
	close(nodeRpc.conns)
	clientPoolLock.Lock()
	if clientPool == nil {
		clientPool = make(map[string]chan *rpc.Client)
	}
	clientPool[ip] = nodeRpc.clients //添加
	clientPoolLock.Unlock()
	logrus.Infof("Create 20 clients (server IP = %s).", ip)
	return nil
}

// 得到可用用户
func getClient(ip string) (*rpc.Client, error) {
	clientPoolLock.RLock()
	clients := clientPool[ip]
	clientPoolLock.RUnlock()
	return <-clients, nil
}

// 归还用户
func returnClient(ip string, client *rpc.Client) error {
	clientPoolLock.RLock()
	clientPool[ip] <- client
	clientPoolLock.RUnlock()
	return nil
}

// 关闭相关连接
func (nodeRpc *NodeRpc) closeConn() {
	for range nodeRpc.conns {
		(<-nodeRpc.conns).Close()
	}
}

// 构建客户端与服务器的连接
func (nodeRpc *NodeRpc) connect(client *rpc.Client) error {
	go client.Call("DHT.Ping", Null{}, &Null{}) //尝试建立客户端与服务器的连接
	conn, err := nodeRpc.listener.Accept()
	if err != nil {
		logrus.Error("Building connection error.")
		return err
	}
	go nodeRpc.server.ServeConn(conn) //开始服务
	nodeRpc.clients <- client
	nodeRpc.conns <- conn
	return nil
}
