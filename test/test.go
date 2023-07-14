package test

import (
	"flag"
	"math/rand"
	"os"
	"time"
)

var (
	help     bool
	testName string
)

func init() {
	flag.BoolVar(&help, "help", false, "help")
	flag.StringVar(&testName, "test", "", "which test(s) do you want to run: basic/advance/all")

	flag.Usage = usage
	flag.Parse()

	if help || (testName != "basic" && testName != "advance" && testName != "all") {
		flag.Usage()
		os.Exit(0)
	}

	rand.Seed(time.Now().UnixNano())
}

func Test() {
	yellow.Printf("Welcome to DHT-2023 Test Program!\n\n")

	var basicFailRate float64
	var forceQuitFailRate float64
	var QASFailRate float64

	switch testName {
	case "all":
		fallthrough
	case "basic":
		yellow.Println("Basic Test Begins:")
		basicPanicked, basicFailedCnt, basicTotalCnt := basicTest()
		if basicPanicked {
			red.Printf("Basic Test Panicked.")
			os.Exit(0)
		}

		basicFailRate = float64(basicFailedCnt) / float64(basicTotalCnt)
		if basicFailRate > basicTestMaxFailRate {
			red.Printf("Basic test failed with fail rate %.4f\n\n", basicFailRate)
		} else {
			green.Printf("Basic test passed with fail rate %.4f\n\n", basicFailRate)
		}

		if testName == "basic" {
			break
		}
		time.Sleep(afterTestSleepTime)
		fallthrough
	case "advance":
		yellow.Println("Advance Test Begins:")

		/* ------ Force Quit Test Begins ------ */
		forceQuitPanicked, forceQuitFailedCnt, forceQuitTotalCnt := forceQuitTest()
		if forceQuitPanicked {
			red.Printf("Force Quit Test Panicked.")
			os.Exit(0)
		}

		forceQuitFailRate = float64(forceQuitFailedCnt) / float64(forceQuitTotalCnt)
		if forceQuitFailRate > forceQuitMaxFailRate {
			red.Printf("Force quit test failed with fail rate %.4f\n\n", forceQuitFailRate)
		} else {
			green.Printf("Force quit test passed with fail rate %.4f\n\n", forceQuitFailRate)
		}
		time.Sleep(afterTestSleepTime)
		/* ------ Force Quit Test Ends ------ */

		/* ------ Quit & Stabilize Test Begins ------ */
		QASPanicked, QASFailedCnt, QASTotalCnt := quitAndStabilizeTest()
		if QASPanicked {
			red.Printf("Quit & Stabilize Test Panicked.")
			os.Exit(0)
		}

		QASFailRate = float64(QASFailedCnt) / float64(QASTotalCnt)
		if QASFailRate > QASMaxFailRate {
			red.Printf("Quit & Stabilize test failed with fail rate %.4f\n\n", QASFailRate)
		} else {
			green.Printf("Quit & Stabilize test passed with fail rate %.4f\n\n", QASFailRate)
		}
		/* ------ Quit & Stabilize Test Ends ------ */
	}

	cyan.Println("\nFinal print:")
	if basicFailRate > basicTestMaxFailRate {
		red.Printf("Basic test failed with fail rate %.4f\n", basicFailRate)
	} else {
		green.Printf("Basic test passed with fail rate %.4f\n", basicFailRate)
	}
	if forceQuitFailRate > forceQuitMaxFailRate {
		red.Printf("Force quit test failed with fail rate %.4f\n", forceQuitFailRate)
	} else {
		green.Printf("Force quit test passed with fail rate %.4f\n", forceQuitFailRate)
	}
	if QASFailRate > QASMaxFailRate {
		red.Printf("Quit & Stabilize test failed with fail rate %.4f\n", QASFailRate)
	} else {
		green.Printf("Quit & Stabilize test passed with fail rate %.4f\n", QASFailRate)
	}
}

func usage() {
	flag.PrintDefaults()
}

// package main

// import (
// 	"crypto/sha1"
// 	"dht/chord"
// 	"fmt"
// 	"math/big"
// 	"time"
// )

// func get_hash(Addr_IP string) *big.Int {
// 	hash := sha1.Sum([]byte(Addr_IP))
// 	hashInt := new(big.Int)
// 	return hashInt.SetBytes(hash[:])
// }

// func main() {

// 	defer func() {
// 		if r := recover(); r != nil {
// 			fmt.Println("Program panicked with", r)
// 		} else {
// 			fmt.Println("Yes")
// 		}
// 	}()
// 	M := 20
// 	var node [20]*chord.Node
// 	for i := 0; i < M; i++ {
// 		node[i] = NewNode(8800 + i)
// 		node[i].Run()
// 	}
// 	time.Sleep(3 * time.Second)
// 	node[0].Create()
// 	time.Sleep(time.Second)
// 	fmt.Println(node[0].Put("a100", "b100"))
// 	node[1].Join(node[0].IP)
// 	time.Sleep(time.Second)
// 	fmt.Println(node[0].Put("a1000", "b1000"))
// 	for i := 1; i < M/2; i++ {
// 		go node[2*i].Join(node[1].IP)
// 		go node[2*i+1].Join(node[0].IP)
// 		time.Sleep(2 * time.Second)
// 		if i == 3 {
// 			fmt.Println(node[1].Put("a1", "b1"))
// 			fmt.Println(node[2].Put("a2", "b2"))
// 			fmt.Println(node[1].Get("a11"))
// 			fmt.Println(node[4].Get("a1"))
// 		}
// 	}
// 	time.Sleep(5 * time.Second)
// 	for i := 0; i < M; i++ {
// 		fmt.Println(node[i].IP, node[i].Predecessor, node[i].Successor_list[0])
// 	}
// 	for i := 0; i < M; i++ {
// 		fmt.Println(get_hash(portToAddr(localAddress, 8800+i)))
// 	}
// 	for i := 0; i < M; i++ {
// 		fmt.Println(node[i].Successor_list)
// 	}
// 	for i := 0; i < M; i++ {
// 		fmt.Println(chord.Ping(node[i].IP))
// 	}
// 	fmt.Println("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
// 	// go node[19].ForceQuit()
// 	// go node[18].ForceQuit()
// 	// go node[17].ForceQuit()
// 	// go node[16].ForceQuit()
// 	// go node[15].ForceQuit()
// 	// time.Sleep(3 * time.Second)
// 	// go node[14].ForceQuit()
// 	// go node[13].ForceQuit()
// 	// go node[12].ForceQuit()
// 	// go node[11].ForceQuit()
// 	// go node[10].ForceQuit()
// 	// time.Sleep(3 * time.Second)
// 	// M -= 10
// 	// for i := 0; i < M; i++ {
// 	// 	fmt.Println(node[i].IP, node[i].Predecessor, node[i].Successor_list[0])
// 	// }
// 	// for i := 0; i < M; i++ {
// 	// 	fmt.Println(get_hash(portToAddr(localAddress, 8800+i)))
// 	// }
// 	// for i := 0; i < M; i++ {
// 	// 	fmt.Println(node[i].Successor_list)
// 	// }
// 	// for i := 0; i < M; i++ {
// 	// 	fmt.Println(chord.Ping(node[i].IP))
// 	// }
// 	// for i := 0; i < M; i++ {
// 	// 	fmt.Println(node[i].Finger)
// 	// }

// 	fmt.Println(node[1].Put("a3", "b3"))
// 	fmt.Println(node[4].Put("a4", "b4"))
// 	fmt.Println(node[1].Get("a11"))
// 	fmt.Println(node[18].Get("a1"))
// 	fmt.Println(node[6].Get("a2"))
// 	fmt.Println(node[8].Get("a3"))
// 	fmt.Println(node[1].Get("a4"))
// 	fmt.Println(node[7].Get("a100"))
// 	fmt.Println(node[17].Get("a1000"))
// 	fmt.Println(node[19].Delete("a1"))
// 	fmt.Println(node[14].Delete("a100000"))
// 	fmt.Println(node[1].Get("a1"))
// 	fmt.Println(node[9].Get("a100000"))
// 	fmt.Println(node[6].Get("a2"))
// 	fmt.Println(node[9].Delete("a2"))
// 	fmt.Println(node[6].Get("a2"))
// 	for i := 0; i < M; i++ {
// 		fmt.Println(i, ":")
// 		node[i].B()
// 	}
// 	// 	// node.Create()
// 	// 	// node := NewNode(8888)
// 	// 	// go node.Run()
// 	// 	// time.Sleep(time.Second)
// 	// 	// node.Create()
// 	// 	// time.Sleep(time.Second)
// }
