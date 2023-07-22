package rpc

import (
	"errors"
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type NodeRpc struct {
	server      *rpc.Server
	listener    net.Listener
	clientPool  map[string]chan *rpc.Client //容纳可用客户端
	DialConns   chan net.Conn               //容纳Dial()产生的连接，在停止服务时关闭
	AcceptConns chan net.Conn               //容纳Accept()产生的连接，在停止服务时关闭
	lock        sync.RWMutex
	listening   bool
}

type Null struct{}

// 远端调用
func (nodeRpc *NodeRpc) RemoteCall(ip string, serviceMethod string, args interface{}, reply interface{}) error {
	if serviceMethod != "DHT.Ping" {
		logrus.Infof("Remote call (server IP = %s, serviceMethod = %s).", ip, serviceMethod)
	}
	if !nodeRpc.listening {
		return errors.New("Offline node server (IP = " + ip + ").")
	}
	client, err := nodeRpc.getClient(ip)
	defer nodeRpc.returnClient(ip, client)
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
	nodeRpc.lock.Lock()
	nodeRpc.clientPool = make(map[string]chan *rpc.Client)
	nodeRpc.DialConns = make(chan net.Conn, 10000)
	nodeRpc.AcceptConns = make(chan net.Conn, 10000)
	nodeRpc.lock.Unlock()
	err = nodeRpc.server.RegisterName(serveName, registerNode) // registerNode是需要注册的节点的指针
	if err != nil {
		logrus.Errorf("Registing error (server IP = %s): %v.", ip, err)
		return err
	} else {
		logrus.Infof("Regist done (server IP = %s, name = %s).", ip, serveName)
	}
	if !nodeRpc.listening {
		nodeRpc.listener, err = net.Listen("tcp", ip)
		if err != nil {
			logrus.Errorf("Listening error (server IP = %s): %v.", ip, err)
			return err
		} else {
			nodeRpc.listening = true
			logrus.Infof("Listen done (server IP = %s, network = tcp).", ip)
		}
		go func() {
			for nodeRpc.listening {
				err := nodeRpc.AcceptAndServe()
				if err != nil {
					logrus.Errorf("Accepting error (server IP = %s): %v.", ip, err)
				}
			}
		}()
	}
	close(start) //疏通开始通道
	select {
	case <-quit:
		nodeRpc.listening = false
		nodeRpc.closeConn() //结束服务
	}
	logrus.Infof("Node stops serving (server IP = %s).", ip)
	nodeRpc.listener.Close()
	return nil
}

// 创建一个节点对应的用户池
func (nodeRpc *NodeRpc) createClient(ip string) error {
	nodeRpc.lock.Lock()
	nodeRpc.clientPool[ip] = make(chan *rpc.Client, 20)
	nodeRpc.lock.Unlock()
	for i := 0; i < 20; i++ {
		conn, err := net.DialTimeout("tcp", ip, time.Second)
		if err != nil {
			logrus.Errorf("Dialing error (server IP = %s): %v.", ip, err)
			continue
		}
		client := rpc.NewClient(conn)
		if nodeRpc.connect(ip, client) != nil {
			logrus.Errorf("Connecting error (server IP = %s): %v.", ip, err)
			continue
		}
		nodeRpc.lock.RLock()
		nodeRpc.DialConns <- conn
		nodeRpc.lock.RUnlock()
	}
	logrus.Infof("Create 20 clients (server IP = %s).", ip)
	return nil
}

// 得到可用用户
func (nodeRpc *NodeRpc) getClient(ip string) (*rpc.Client, error) {
	nodeRpc.lock.RLock()
	clients, ok := nodeRpc.clientPool[ip]
	nodeRpc.lock.RUnlock()
	if !ok {
		err := nodeRpc.createClient(ip)
		if err != nil {
			logrus.Errorf("Create clients error (server IP = %s): %v", ip, err)
			return nil, err
		}
		nodeRpc.lock.RLock()
		clients, ok = nodeRpc.clientPool[ip]
		nodeRpc.lock.RUnlock()
	}
	return <-clients, nil
}

// 归还用户
func (nodeRpc *NodeRpc) returnClient(ip string, client *rpc.Client) error {
	nodeRpc.lock.RLock()
	nodeRpc.clientPool[ip] <- client
	nodeRpc.lock.RUnlock()
	return nil
}

// 关闭相关连接
func (nodeRpc *NodeRpc) closeConn() {
	nodeRpc.lock.Lock()
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
	nodeRpc.lock.Unlock()
}

// 构建客户端与服务器的连接
func (nodeRpc *NodeRpc) connect(ip string, client *rpc.Client) error {
	done := make(chan error)
	go func() {
		done <- client.Call("DHT.Ping", Null{}, &Null{}) //尝试建立客户端与服务器的连接
	}()
	err := <-done
	if err != nil {
		return err
	}
	nodeRpc.lock.RLock()
	nodeRpc.clientPool[ip] <- client
	nodeRpc.lock.RUnlock()
	return nil
}

func (nodeRpc *NodeRpc) AcceptAndServe() error {
	conn, err := nodeRpc.listener.Accept()
	if err != nil {
		logrus.Errorf("Building connection error: %v", err)
		return err
	}
	go nodeRpc.server.ServeConn(conn) //开始服务
	nodeRpc.lock.RLock()
	nodeRpc.AcceptConns <- conn
	nodeRpc.lock.RUnlock()
	return nil
}
