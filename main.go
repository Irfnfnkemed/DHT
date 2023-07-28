package main

import (
	"dht/test"
	"flag"
	"math/rand"
	"os"
	"time"
)

func init() {
	flag.BoolVar(&test.Help, "help", false, "help")
	flag.StringVar(&test.ProtocolTest, "protocol", "", "which protocol do you want to run: naive/chord/kademlia")
	flag.StringVar(&test.TestName, "test", "", "which test(s) do you want to run: basic/advance/all")
	flag.Usage = test.Usage
	flag.Parse()
	if test.Help || (test.ProtocolTest != "naive" && test.ProtocolTest != "chord" && test.ProtocolTest != "kademlia") {
		flag.Usage()
		os.Exit(0)
	}
	test.SetProtocol(test.ProtocolTest)
	if test.Help || (test.TestName != "basic" && test.TestName != "advance" && test.TestName != "all") {
		flag.Usage()
		os.Exit(0)
	}

	rand.Seed(time.Now().UnixNano())
}

func main() {
	test.Test()
	// node := new(chat.ChatNode)
	// chat.PrintCentre("Please type your name:", "yellow")
	// userName := chat.Scan('\n')
	// chat.PrintCentre("Please type your IP:", "yellow")
	// userIp := chat.Scan('\n')
	// chat.PrintCentre("Creat/Join?", "yellow")
	// chat.PrintCentre("Type Y(y)/N(n):", "yellow")
	// tmp := chat.Scan('\n')
	// flag := (tmp == "Y" || tmp == "y")
	// if flag {
	// 	err := node.Login(userName, userIp, "")
	// 	if err != nil {
	// 		chat.PrintCentre(err.Error(), "red")
	// 	} else {
	// 		chat.PrintCentre("Successfully log in.", "yellow")
	// 	}
	// } else {
	// 	chat.PrintCentre("Please type the node IP (the node is in the P2P chat system):", "yellow")
	// 	enterIp := chat.Scan('\n')
	// 	err := node.Login(userName, userIp, enterIp)
	// 	if err != nil {
	// 		chat.PrintCentre(err.Error(), "red")
	// 	} else {
	// 		chat.PrintCentre("Successfully log in.", "yellow")
	// 	}
	// }
	// chat.PrintCentre("Type anything to continue.", "white")
	// chat.Scan('\n')
	// node.Interactive()
	// node.LogOut()
}
