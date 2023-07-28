package main

import (
	"bufio"
	"dht/chat"
	"dht/test"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	color.New(color.FgYellow).Println("Test/Chat?\nType:test/chat:")
	for {
		reader := bufio.NewReader(os.Stdin)
		control, _ := reader.ReadString('\n')
		control = strings.TrimRight(control, string('\n'))
		if control == "test" {
			test.Test()
			return
		} else if control == "chat" {
			chat.Chat()
			return
		} else {
			color.New(color.FgRed).Println("Name error! Please type again.")
		}
	}
}
