package chord

import (
	"net"
	"net/rpc"
	"time"

	"github.com/sirupsen/logrus"
)

func Remote_call(ip string, service_method string, args interface{}, reply interface{}) error {
	logrus.Infof("Remote call (server IP = %s).", ip)
	conn, err := net.DialTimeout("tcp", ip, time.Second)
	if err != nil {
		logrus.Errorf("Dailing error (server IP = %s): %v.", ip, err)
		return err
	}
	client := rpc.NewClient(conn)
	err = client.Call(service_method, args, reply)
	defer func() { conn.Close() }()
	if err != nil {
		logrus.Errorf("Calling error (server IP = %s): %v.", ip, err)
		return err
	}
	return nil
}

func (node *Node) Serve() error {
	var err error = nil
	node.Server = rpc.NewServer()
	err = node.Server.RegisterName("DHT", RPC_wrapper{node})
	if err != nil {
		logrus.Errorf("Registing error (server IP = %s): %v.", node.This.IP, err)
		return err
	}
	node.Listener, err = net.Listen("tcp", node.This.IP)
	if err != nil {
		logrus.Errorf("Listening error (server IP = %s): %v.", node.This.IP, err)
		return err
	}
	for {
		conn, err := node.Listener.Accept()
		if err != nil {
			logrus.Errorf("Accepting error (server IP = %s): %v.", node.This.IP, err)
			continue
		}
		go rpc.ServeConn(conn)
	}
	return nil
}
