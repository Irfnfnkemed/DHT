package chord

import (
	"net"
	"net/rpc"
	"time"

	"github.com/sirupsen/logrus"
)

func Remote_call(ip string, service_method string, args interface{}, reply interface{}) error {
	logrus.Infof("Remote call (server IP = %s, service_method = %s).", ip, service_method)
	conn, err := net.DialTimeout("tcp", ip, time.Second)
	if err != nil {
		logrus.Errorf("Dailing error (server IP = %s): %v.", ip, err)
		return err
	} else {
		logrus.Infof("Dail done (server IP = %s).", ip)
	}
	client := rpc.NewClient(conn)
	err = client.Call(service_method, args, reply)
	defer func() { conn.Close() }()
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
	node.Server = rpc.NewServer()
	err = node.Server.RegisterName("DHT", &RPC_wrapper{node})
	if err != nil {
		logrus.Errorf("Registing error (server IP = %s): %v.", node.IP, err)
		return err
	} else {
		logrus.Infof("Regist done (server IP = %s, name = DHT).", node.IP)
	}
	node.Listener, err = net.Listen("tcp", node.IP)
	if err != nil {
		logrus.Errorf("Listening error (server IP = %s): %v.", node.IP, err)
		return err
	} else {
		logrus.Infof("Listen done (server IP = %s, network = tcp).", node.IP)
	}
	for node.Online {
		conn, err := node.Listener.Accept()
		if err != nil {
			logrus.Errorf("Accepting error (server IP = %s): %v.", node.IP, err)
			continue
		}
		go node.Server.ServeConn(conn)
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}
