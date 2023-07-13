package chord

import (
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var client_pool map[string]chan *rpc.Client //ip到用户池的映射
var client_pool_lock sync.RWMutex

type Node_rpc struct {
	server   *rpc.Server
	listener net.Listener
	clients  chan *rpc.Client //容量为20，容纳可用客户端
	conns    chan net.Conn    //容量为40，容纳DIal()、Accept()产生的连接，在停止服务时关闭
}

// 远端调用
func Remote_call(ip string, service_method string, args interface{}, reply interface{}) error {
	logrus.Infof("Remote call (server IP = %s, service_method = %s).", ip, service_method)
	client, err := get_client(ip)
	defer return_client(ip, client)
	if err != nil {
		logrus.Errorf("Getting client error (server IP = %s): %v.", ip, err)
		return err
	}
	err = client.Call(service_method, args, reply)
	if err != nil {
		logrus.Errorf("Calling error (server IP = %s): %v.", ip, err)
		return err
	} else {
		logrus.Infof("Call done (server IP = %s, service_method = %s).", ip, service_method)
	}
	return nil
}

// 节点服务
func (node *Node) Serve() error {
	var err error = nil
	node.RPC.server = rpc.NewServer()
	err = node.RPC.server.RegisterName("DHT", &RPC_wrapper{node})
	if err != nil {
		logrus.Errorf("Registing error (server IP = %s): %v.", node.IP, err)
		return err
	} else {
		logrus.Infof("Regist done (server IP = %s, name = DHT).", node.IP)
	}
	node.RPC.listener, err = net.Listen("tcp", node.IP)
	if err != nil {
		logrus.Errorf("Listening error (server IP = %s): %v.", node.IP, err)
		return err
	} else {
		logrus.Infof("Listen done (server IP = %s, network = tcp).", node.IP)
	}
	err = node.RPC.create_client(node.IP)
	if err != nil {
		return err
	}
	close(node.start) //疏通开始通道
	select {
	case <-node.quit:
		node.RPC.close_conn() //结束服务
	}
	logrus.Infof("Node stops serving (server IP = %s).", node.IP)
	node.RPC.close_conn()
	node.RPC.listener.Close()
	return nil
}

// 测试节点是否上线
func Ping(ip string) bool {
	client, err := get_client(ip)
	defer return_client(ip, client)
	if err != nil {
		return false
	}
	err = client.Call("DHT.Ping", Null{}, &Null{})
	return err == nil
}

// 创建一个节点的用户池
func (node_rpc *Node_rpc) create_client(ip string) error {
	node_rpc.clients = make(chan *rpc.Client, 20)
	node_rpc.conns = make(chan net.Conn, 40)
	for i := 0; i < 20; i++ {
		conn, err := net.DialTimeout("tcp", ip, time.Second)
		if err != nil {
			logrus.Errorf("Dialing error (server IP = %s): %v.", ip, err)
			continue
		}
		client := rpc.NewClient(conn)
		if node_rpc.connect(client) != nil {
			logrus.Errorf("Connecting error (server IP = %s): %v.", ip, err)
			continue
		}
		node_rpc.conns <- conn
	}
	close(node_rpc.conns)
	client_pool_lock.Lock()
	if client_pool == nil {
		client_pool = make(map[string]chan *rpc.Client)
	}
	client_pool[ip] = node_rpc.clients //添加
	client_pool_lock.Unlock()
	logrus.Infof("Create 20 clients (server IP = %s).", ip)
	return nil
}

// 得到可用用户
func get_client(ip string) (*rpc.Client, error) {
	client_pool_lock.RLock()
	clients := client_pool[ip]
	client_pool_lock.RUnlock()
	return <-clients, nil
}

// 归还用户
func return_client(ip string, client *rpc.Client) error {
	client_pool_lock.RLock()
	client_pool[ip] <- client
	client_pool_lock.RUnlock()
	return nil
}

// 关闭相关连接
func (node_rpc *Node_rpc) close_conn() {
	for range node_rpc.conns {
		(<-node_rpc.conns).Close()
	}
}

// 构建客户端与服务器的连接
func (node_rpc *Node_rpc) connect(client *rpc.Client) error {
	go client.Call("DHT.Ping", Null{}, &Null{}) //尝试建立客户端与服务器的连接
	conn, err := node_rpc.listener.Accept()
	if err != nil {
		logrus.Error("Building connection error.")
		return err
	}
	go node_rpc.server.ServeConn(conn) //开始服务
	node_rpc.clients <- client
	node_rpc.conns <- conn
	return nil
}
