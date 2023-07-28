package test

import (
	"bufio"
	"os"
	"strings"
	"time"
)

var testName string

func Test() {
	yellow.Println("Which protocol do you want to test?\nType naive/chord/kademlia")
	for {
		reader := bufio.NewReader(os.Stdin)
		protocol, _ := reader.ReadString('\n')
		protocol = strings.TrimRight(protocol, string('\n'))
		if protocol != "naive" && protocol != "chord" && protocol != "kademlia" {
			red.Println("Protocol name error! Please type again.")
		} else {
			SetProtocol(protocol)
			break
		}
	}
	yellow.Println("Which part do you want to test?\nType basic/advance/all")
	for {
		reader := bufio.NewReader(os.Stdin)
		part, _ := reader.ReadString('\n')
		part = strings.TrimRight(part, string('\n'))
		if part != "basic" && part != "advance" && part != "all" {
			red.Println("Part name error! Please type again.")
		} else {
			testName = part
			break
		}
	}
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
