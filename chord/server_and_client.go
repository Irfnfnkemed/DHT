package chord

import (
	"errors"
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var client_pool sync.Map

type Node_rpc struct {
	server   *rpc.Server
	listener net.Listener
	clients  chan *rpc.Client //容量为20
}

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

func (node *Node) Serve() error {
	var err error = nil
	node.Server.server = rpc.NewServer()
	err = node.Server.server.RegisterName("DHT", &RPC_wrapper{node})
	if err != nil {
		logrus.Errorf("Registing error (server IP = %s): %v.", node.IP, err)
		return err
	} else {
		logrus.Infof("Regist done (server IP = %s, name = DHT).", node.IP)
	}
	node.Server.listener, err = net.Listen("tcp", node.IP)
	if err != nil {
		logrus.Errorf("Listening error (server IP = %s): %v.", node.IP, err)
		return err
	} else {
		logrus.Infof("Listen done (server IP = %s, network = tcp).", node.IP)
	}
	err = node.Server.create_client(node.IP)
	if err != nil {
		return err
	}
	for node.Online {
		conn, err := node.Server.listener.Accept()
		if err != nil {
			logrus.Errorf("Accepting error (server IP = %s): %v.", node.IP, err)
			continue
		}
		go node.Server.server.ServeConn(conn)
		time.Sleep(200 * time.Microsecond)
	}
	return nil
}

func Ping(ip string) bool {
	client, err := get_client(ip)
	defer return_client(ip, client)
	if err != nil {
		return false
	}
	err = client.Call("DHT.Ping", Null{}, &Null{})
	return err == nil
}

func (node_rpc *Node_rpc) create_client(ip string) error {
	node_rpc.clients = make(chan *rpc.Client, 20)
	for i := 0; i < 20; i++ {
		conn, err := net.DialTimeout("tcp", ip, time.Second)
		if err != nil {
			logrus.Errorf("Dialing error (server IP = %s): %v.", ip, err)
			i--
			continue
		}
		node_rpc.clients <- rpc.NewClient(conn)
	}
	client_pool.Store(ip, node_rpc.clients) //添加
	logrus.Infof("Create 20 clients (server IP = %s).", ip)
	return nil
}

func get_client(ip string) (*rpc.Client, error) {
	clients_tmp, ok := client_pool.Load(ip)
	if !ok {
		return nil, errors.New("Get client error.")
	} else {
		clients, _ := clients_tmp.(chan *rpc.Client)
		return <-clients, nil
	}
}

func return_client(ip string, client *rpc.Client) error {
	clients_tmp, ok := client_pool.Load(ip)
	if !ok {
		return errors.New("Return client error.")
	} else {
		clients, _ := clients_tmp.(chan *rpc.Client)
		clients <- client
		return nil
	}
}
