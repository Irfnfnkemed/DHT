package rpc

import (
	"errors"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type NodeRpc struct {
	server      *rpc.Server
	listener    net.Listener
	clientPool  map[string]chan *rpc.Client //容纳可用客户端
	DialConns   chan net.Conn               //容纳Dial()产生的连接，在停止服务时关闭
	AcceptConns chan net.Conn               //容纳Accept()产生的连接，在停止服务时关闭
	clientLock  sync.RWMutex
	connLock    sync.RWMutex
	listening   bool
	serveName   []string
}

type Null struct{}

const CallTimeOut = 10 * time.Second
const PendClientTimeOut = 500 * time.Millisecond

// 为节点类registerNode注册rpc服务，其中第一个注册的服务应该有Ping函数
func (nodeRpc *NodeRpc) Register(serveName string, registerNode interface{}) error {
	if nodeRpc.server == nil {
		nodeRpc.server = rpc.NewServer()
	}
	if nodeRpc.serveName == nil {
		nodeRpc.serveName = make([]string, 0)
	}
	nodeRpc.serveName = append(nodeRpc.serveName, serveName)
	err := nodeRpc.server.RegisterName(serveName, registerNode) // registerNode是需要注册的节点的指针
	if err != nil {
		//logrus.Errorf("Registing error: %v.", err)
		return err
	} else {
		//logrus.Infof("Regist done (name = %s).", serveName)
		return nil
	}
}

// 节点服务
func (nodeRpc *NodeRpc) Serve(ip string, start, quit chan bool) error {
	var err error = nil
	nodeRpc.clientLock.Lock()
	nodeRpc.connLock.Lock()
	nodeRpc.clientPool = make(map[string]chan *rpc.Client)
	nodeRpc.DialConns = make(chan net.Conn, 10000)
	nodeRpc.AcceptConns = make(chan net.Conn, 10000)
	nodeRpc.clientLock.Unlock()
	nodeRpc.connLock.Unlock()
	nodeRpc.listener, err = net.Listen("tcp", ip)
	if err != nil {
		//logrus.Errorf("Listening error (server IP = %s): %v.", ip, err)
		return err
	} else {
		nodeRpc.listening = true
		//logrus.Infof("Listen done (server IP = %s, network = tcp).", ip)
	}
	go func() {
		for nodeRpc.listening {
			err := nodeRpc.Accept()
			if err != nil {
				//logrus.Errorf("Accepting error (server IP = %s): %v.", ip, err)
			}
		}
	}()
	close(start) //疏通开始通道
	select {
	case <-quit:
		nodeRpc.listening = false
		nodeRpc.listener.Close()
		nodeRpc.closeConn() //结束服务
	}
	//logrus.Infof("Node stops serving (server IP = %s).", ip)
	return nil
}

// 远端调用
func (nodeRpc *NodeRpc) RemoteCall(ip string, serviceMethod string, args interface{}, reply interface{}) error {
	//logrus.Infof("Remote call (server IP = %s, serviceMethod = %s).", ip, serviceMethod)
	if !nodeRpc.listening {
		return errors.New("Offline node server (IP = " + ip + ").")
	}
	client, err := nodeRpc.getClient(ip)
	defer nodeRpc.returnClient(ip, client)
	if err != nil {
		//logrus.Errorf("Getting client error (server IP = %s): %v.", ip, err)
		if client != nil {
			client.Close()
			client = nil
		}
		return err
	}
	done := make(chan error, 1)
	go func() {
		done <- client.Call(serviceMethod, args, reply)
	}()
	select {
	case err := <-done:
		if err != nil {
			//logrus.Errorf("Calling error (server IP = %s): %v.", ip, err)
			return err
		} else {
			//logrus.Infof("Call done (server IP = %s, serviceMethod = %s).", ip, serviceMethod)
			return nil
		}
	case <-time.After(CallTimeOut):
		if client != nil {
			client.Close()
			client = nil
		}
		return errors.New("Call time out.") // 超时
	}
}

// 得到可用用户
func (nodeRpc *NodeRpc) getClient(ip string) (*rpc.Client, error) {
	nodeRpc.clientLock.RLock()
	clients, ok := nodeRpc.clientPool[ip]
	nodeRpc.clientLock.RUnlock()
	if !ok {
		err := nodeRpc.createClient(ip)
		if err != nil {
			//logrus.Errorf("Create clients error (server IP = %s): %v", ip, err)
			return nil, err
		}
		nodeRpc.clientLock.RLock()
		clients, ok = nodeRpc.clientPool[ip]
		nodeRpc.clientLock.RUnlock()
	}
	if clients == nil {
		return nil, errors.New("Offline node.")
	}
	select {
	case client := <-clients:
		if client == nil {
			nodeRpc.deleteClients(ip)
			return nil, errors.New("Offline node.")
		}
		return client, nil
	case <-time.After(PendClientTimeOut):
		nodeRpc.deleteClients(ip)
		return nil, errors.New("Get client time out.") // 超时
	}
}

// 创建nodeRpc至ip节点的连接池
func (nodeRpc *NodeRpc) createClient(ip string) error {
	nodeRpc.clientLock.Lock()
	nodeRpc.clientPool[ip] = make(chan *rpc.Client, 50)
	nodeRpc.clientLock.Unlock()
	flag := false
	for i := 0; i < 10; i++ {
		conn, err := net.DialTimeout("tcp", ip, time.Second)
		if err != nil {
			//logrus.Errorf("Dialing error (server IP = %s): %v.", ip, err)
			break
		}
		client := rpc.NewClient(conn)
		err = nodeRpc.connect(ip, client)
		if err != nil {
			//logrus.Errorf("Connecting error (server IP = %s): %v.", ip, err)
			break
		}
		flag = true
		nodeRpc.connLock.Lock()
		nodeRpc.DialConns <- conn
		nodeRpc.connLock.Unlock()
	}
	if flag {
		//logrus.Infof("Create 10 clients (server IP = %s).", ip)
	} else {
		nodeRpc.deleteClients(ip)
		return errors.New("Creating clients error.")
	}
	return nil
}

// 归还用户
func (nodeRpc *NodeRpc) returnClient(ip string, client *rpc.Client) error {
	if client == nil {
		nodeRpc.deleteClients(ip)
		return nil
	}
	nodeRpc.clientLock.Lock()
	clients, ok := nodeRpc.clientPool[ip]
	if ok {
		clients <- client
	} else {
		client.Close()
	}
	nodeRpc.clientLock.Unlock()
	return nil
}

// 关闭相关连接
func (nodeRpc *NodeRpc) closeConn() {
	nodeRpc.connLock.Lock()
	close(nodeRpc.DialConns)
	for range nodeRpc.DialConns {
		conn := <-nodeRpc.DialConns
		if conn != nil {
			conn.Close()
		}
	}
	nodeRpc.DialConns = nil
	close(nodeRpc.AcceptConns)
	for range nodeRpc.AcceptConns {
		conn := <-nodeRpc.AcceptConns
		if conn != nil {
			conn.Close()
		}
	}
	nodeRpc.AcceptConns = nil
	nodeRpc.connLock.Unlock()
	nodeRpc.clientLock.Lock()
	for _, clients := range nodeRpc.clientPool {
		if clients != nil {
			close(clients)
			for range clients {
				client := <-clients
				if client != nil {
					client.Close()
				}
			}
		}
	}
	nodeRpc.clientPool = make(map[string]chan *rpc.Client) //清空，防止再次关闭等异常操作
	nodeRpc.clientLock.Unlock()
}

// 构建客户端与服务器的连接
func (nodeRpc *NodeRpc) connect(ip string, client *rpc.Client) error {
	if client == nil {
		return errors.New("Empty Client.")
	}
	done := make(chan error, 1)
	go func() {
		done <- client.Call(nodeRpc.serveName[0]+".Ping", Null{}, &Null{}) //尝试建立客户端与服务器的连接
	}()
	select {
	case err := <-done:
		if err != nil {
			if client != nil {
				client.Close()
			}
			return err
		}
		nodeRpc.clientLock.Lock()
		nodeRpc.clientPool[ip] <- client
		nodeRpc.clientLock.Unlock()
		return nil
	case <-time.After(CallTimeOut):
		if client != nil {
			client.Close()
		}
		return errors.New("Build connection time out.") // 超时
	}
}

func (nodeRpc *NodeRpc) Accept() error {
	conn, err := nodeRpc.listener.Accept()
	if err != nil {
		//logrus.Errorf("Building connection error: %v", err)
		return err
	}
	go nodeRpc.server.ServeConn(conn) //开始服务
	nodeRpc.connLock.Lock()
	nodeRpc.AcceptConns <- conn
	nodeRpc.connLock.Unlock()
	return nil
}

func (nodeRpc *NodeRpc) deleteClients(ip string) {
	nodeRpc.clientLock.Lock()
	clients, ok := nodeRpc.clientPool[ip]
	if ok && clients != nil {
		close(clients)
		for range clients {
			client := <-clients
			if client != nil {
				client.Close()
			}
		}
		delete(nodeRpc.clientPool, ip)
	}
	nodeRpc.clientLock.Unlock()
}
