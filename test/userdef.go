package test

import (
	"dht/chord"
	"dht/kademlia"
	"dht/naive"
	"errors"
)

/*
 * In this file, you need to implement the "NewNode" function.
 * This function should create a new DHT node and return it.
 * You can use the "naive.Node" struct as a reference to implement your own struct.
 */

var Protocol string

func SetProtocol(protocol string) error {
	if protocol != "naive" && protocol != "chord" && protocol != "kademlia" {
		return errors.New("Protocol name false.")
	} else {
		Protocol = protocol
		return nil
	}
}

func NewNode(port int) dhtNode {
	// Todo: create a node and then return it.
	switch Protocol {
	case "naive":
		node := new(naive.Node)
		node.Init(portToAddr(localAddress, port))
		return node
	case "chord":
		node := new(chord.Node)
		node.Init(portToAddr(localAddress, port))
		return node
	case "kademlia":
		node := new(kademlia.Node)
		node.Init(portToAddr(localAddress, port))
		return node
	}
	return nil
}
