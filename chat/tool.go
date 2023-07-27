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

func getHash(ip string) *big.Int {
	hash := sha1.Sum([]byte(ip))
	hashInt := new(big.Int)
	return hashInt.SetBytes(hash[:])
}

func randString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func getGroupIp(groupSeed string, groupStartTime, sendTime time.Time) string {
	time := int(sendTime.Sub(groupStartTime).Minutes() / 60.0) //每一小时更换一次id
	return groupSeed + strconv.Itoa(time)
}

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

func PrintCentre(message, colorName string) {
	fmtColor := seclecColor(colorName)
	fmtColor.Println(strings.Repeat(" ", (consoleWidth-len(message))/2) + message) //生成填充字符串，加到左侧
}

func Scan(separator byte) string {
	reader := bufio.NewReader(os.Stdin)
	name, _ := reader.ReadString(separator)
	return strings.TrimRight(name, string(separator))
}

func seclecColor(colorName string) *color.Color {
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
	return fmtColor
}

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
