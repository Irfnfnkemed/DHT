package chat

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"golang.org/x/sys/unix"
)

var (
	letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

var (
	blue    = color.New(color.FgBlue)
	green   = color.New(color.FgGreen)
	red     = color.New(color.FgRed)
	magenta = color.New(color.FgMagenta)
	yellow  = color.New(color.FgYellow)
	cyan    = color.New(color.FgCyan)
	white   = color.New(color.FgWhite)
)

var consoleWidth int
var flushPadding string
var separator string
var cursorPadding string
var cursor string
var upArrow string
var downArrow string
var leftArrow string
var rightArrow string

// 得到控制台宽度，便于居中打印
func setConsoleWidth() error {
	console := os.Stdout
	if console == nil {
		consoleWidth = 0
		return errors.New("unable to access the console")
	}
	fd := console.Fd()
	ws, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	if err != nil {
		consoleWidth = 0
		return err
	}
	consoleWidth = int(ws.Col)
	return nil
}

// 初始化一些页面打印组件
func setStrings() {
	flushPadding = strings.Repeat("\n", 50) + strings.Repeat("-", consoleWidth) + "\n"
	separator = "\n" + strings.Repeat("-", consoleWidth) + "\n"
	cursorPadding = "    "
	cursor = "--> "
	upArrow = fmt.Sprintf("\x1B\x5B\x41")
	downArrow = fmt.Sprintf("\x1B\x5B\x42")
	leftArrow = fmt.Sprintf("\x1B\x5B\x44")
	rightArrow = fmt.Sprintf("\x1B\x5B\x43")
}

// 打印Logo
func printLogo() {
	PrintCentre("__         __", "yellow")
	PrintCentre("/ /___   ___\\ \\", "yellow")
	PrintCentre("/  ___/   \\___  \\", "yellow")
	PrintCentre("/_____/     \\_____\\", "yellow")
	fmt.Print("\n")
}

func printConsole(len, cursorIndex int, consoleCommand []string) {
	for i := 0; i < len; i++ {
		if i == cursorIndex {
			cyan.Println(cursor + consoleCommand[i])
		} else {
			fmt.Println(cursorPadding + consoleCommand[i])
		}
	}
}

func printConsoleSelected(len, cursorIndex int, consoleCommand []string, selected bool) {
	for i := 0; i < len; i++ {
		if i == cursorIndex {
			if selected {
				fmt.Println(cursor + consoleCommand[i])
			} else {
				cyan.Println(cursor + consoleCommand[i])
			}
		} else {
			fmt.Println(cursorPadding + consoleCommand[i])
		}
	}
}

// 得到hash值
func getHash(ip string) *big.Int {
	hash := sha1.Sum([]byte(ip))
	hashInt := new(big.Int)
	return hashInt.SetBytes(hash[:])
}

// 得到随机字符串
func randString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// 得到某时刻下群聊对应的ID
func getGroupIp(groupSeed string, groupStartTime, sendTime time.Time) string {
	time := int(sendTime.Sub(groupStartTime).Minutes() / groupIdValidTime.Minutes()) //每隔groupIdValidTime更换一次id
	return groupSeed + strconv.Itoa(time)
}

// 将json串转换为[]InfoRecord
func parseToInfoRecord(data string) ([]InfoRecord, error) {
	infos := []InfoRecord{}
	parts := strings.Split(data, "\n")
	for _, part := range parts {
		if part == "" {
			continue
		}
		infoRecord := InfoRecord{}
		part = strings.TrimRight(part, "\n")
		err := json.Unmarshal([]byte(part), &infoRecord)
		if err != nil {
			return infos, err
		}
		infos = append(infos, infoRecord)
	}
	return infos, nil
}

// 将[]InfoRecord打印在交互页面上
func (chatNode *ChatNode) printChatRecords(infos []InfoRecord) {
	if len(infos) == 0 {
		return
	}
	timeString := infos[0].SendTime.Format("2006-01-02 15:04")
	PrintCentre(timeString, "white")
	for _, infoRecord := range infos {
		tmpTimeString := infoRecord.SendTime.Format("2006-01-02 15:04")
		if tmpTimeString != timeString {
			timeString = tmpTimeString
			PrintCentre(timeString, "white")
		}
		if infoRecord.FromName == chatNode.name {
			yellow.Print(infoRecord.FromName)
		} else {
			green.Print(infoRecord.FromName)
		}
		blue.Print(" >>>\n   ")
		fmt.Println(infoRecord.Info)
	}
}

// 指定颜色的居中打印
func PrintCentre(message, colorName string) {
	var fmtColor *color.Color
	switch colorName {
	case "blue":
		fmtColor = blue
	case "green":
		fmtColor = green
	case "red":
		fmtColor = red
	case "magenta":
		fmtColor = magenta
	case "yellow":
		fmtColor = yellow
	case "cyan":
		fmtColor = cyan
	case "white":
		fmtColor = white
	}
	fmtColor.Println(strings.Repeat(" ", (consoleWidth-len(message))/2) + message) //生成填充字符串，加到左侧
}

// 读入一行(去除末尾的'\n'及空格)
func Scan(separator byte) string {
	reader := bufio.NewReader(os.Stdin)
	name, _ := reader.ReadString(separator)
	return strings.TrimRight(name, string(separator))
}

// 根据输入控制符得到光标位置
func moveCursor(len, cursorIndex int, control string) int {
	if control == upArrow {
		if cursorIndex > 0 {
			return cursorIndex - 1
		} else {
			return len - 1
		}
	} else if control == downArrow {
		if cursorIndex < len-1 {
			return cursorIndex + 1
		} else {
			return 0
		}
	} else {
		return cursorIndex
	}
}

// 得到用户账号的IP和上线情况
func (chatNode *ChatNode) getUserAccount(friendName string) (string, bool, error) {
	friendAccount := AccountRecord{}
	ok, friendAccountString := chatNode.node.Get(friendName)
	if !ok {
		return "", false, errors.New("Not existed user!")
	}
	err := json.Unmarshal([]byte(friendAccountString), &friendAccount)
	if err != nil {
		return "", false, errors.New("Parse error!")
	}
	return friendAccount.IP, friendAccount.Online, nil
}

// 输入Y/y，返回true；输入N/n，返回false
func getSelection() bool {
	for {
		control := Scan('\n')
		if control == "Y" || control == "y" {
			return true
		} else if control == "N" || control == "n" {
			return false
		}
	}
}
