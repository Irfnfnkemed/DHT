package chat

import (
	"fmt"
	"strconv"
	"time"
)

// 主程序
func Chat() {
	node := new(ChatNode)
	PrintCentre("Please type your name:", "yellow")
	userName := Scan('\n')
	PrintCentre("Please type your IP:", "yellow")
	userIp := Scan('\n')
	PrintCentre("Creat/Join?", "yellow")
	PrintCentre("Type Y(y)/N(n):", "yellow")
	tmp := Scan('\n')
	flag := (tmp == "Y" || tmp == "y")
	if flag {
		err := node.Login(userName, userIp, "")
		if err != nil {
			PrintCentre(err.Error(), "red")
		} else {
			PrintCentre("Successfully log in.", "green")
		}
	} else {
		PrintCentre("Please type the node IP (the node is in the P2P chat system):", "yellow")
		enterIp := Scan('\n')
		err := node.Login(userName, userIp, enterIp)
		if err != nil {
			PrintCentre(err.Error(), "red")
		} else {
			PrintCentre("Successfully log in.", "green")
		}
	}
	PrintCentre("Type anything to continue.", "white")
	Scan('\n')
	if flag {
		node.interactive()
		node.LogOut()
	} else {
		node.help()
	}
}

// 交互页面间的状态转移
func (chatNode *ChatNode) interactive() {
	next := "homePage"
	for {
		switch next {
		case "homePage":
			next = chatNode.homePage()
		case "viewFriendList":
			next = chatNode.viewFriendList()
		case "viewGroupChatList":
			next = chatNode.viewGroupChatList()
		case "viewFriendRequest":
			next = chatNode.viewFriendRequest()
		case "viewGroupChatInvitation":
			next = chatNode.viewGroupChatInvitation()
		case "addNewFriend":
			next = chatNode.addNewFriend()
		case "help":
			next = chatNode.help()
		case "exit":
			chatNode.exit()
			return
		}
	}
}

// 首页
func (chatNode *ChatNode) homePage() string {
	cursorIndex := 0
	consoleCommand := []string{"View friend list", "View group chat list", "View friend request", "View group chat invitation", "Add new friend", "help", "exit"}
	for {
		fmt.Println(flushPadding)
		PrintCentre(chatNode.name+", welcome to WGJ's P2P chat!", "yellow")
		fmt.Println(separator)
		PrintCentre("Press up/down arrow to move the cursor.", "magenta")
		PrintCentre("Press enter to confirm.", "magenta")
		for i := 0; i < 7; i++ {
			if cursorIndex == i {
				blue.Print(cursor)
			} else {
				fmt.Print(cursorPadding)
			}
			fmt.Println(consoleCommand[i])
		}
		control := Scan('\n')
		if control == "" {
			if cursorIndex == 0 {
				return "viewFriendList"
			} else if cursorIndex == 1 {
				return "viewGroupChatList"
			} else if cursorIndex == 2 {
				return "viewFriendRequest"
			} else if cursorIndex == 3 {
				return "viewGroupChatInvitation"
			} else if cursorIndex == 4 {
				return "addNewFriend"
			} else if cursorIndex == 5 {
				return "help"
			} else if cursorIndex == 6 {
				return "exit"
			}
		} else {
			cursorIndex = moveCursor(7, cursorIndex, control)
		}
	}
}

// 查看好友列表
func (chatNode *ChatNode) viewFriendList() string {
	friendList := chatNode.GetFriendList()
	friendCursorIndex := 0
	friendCursorLen := len(friendList)
	consoleCursorIndex := 0
	selectedFriend := make([]bool, friendCursorLen)
	selectFriendCursor := false
	selectedFriendNum := 0
	consoleCommand := []string{"Invite selected friends to group chat", "Back to home page", "help"}
	for {
		fmt.Println(flushPadding)
		yellow.Println("Friend list:")
		if len(friendList) == 0 {
			PrintCentre("You don't have any friends now!", "red")
		} else {
			for i, name := range friendList {
				if i == friendCursorIndex {
					if selectFriendCursor {
						blue.Print(cursor)
					} else {
						fmt.Print(cursor)
					}
				} else {
					if selectedFriend[i] {
						green.Print(cursor)
					} else {
						fmt.Print(cursorPadding)
					}
				}
				if selectedFriend[i] {
					green.Println(name)
				} else {
					yellow.Println(name)
				}
			}
		}
		fmt.Println(separator)
		PrintCentre("Press up/down arrow to move the cursor.", "magenta")
		PrintCentre("Press left/right arrow to get the friendList/console cursor.", "magenta")
		PrintCentre("Select friends through friendList cursor.", "magenta")
		PrintCentre("Press enter to confirm.", "magenta")
		PrintCentre("Press Tab clear the friends selections.", "magenta")
		for i := 0; i < 3; i++ {
			if consoleCursorIndex == i {
				if selectFriendCursor {
					fmt.Print(cursor)
				} else {
					blue.Print(cursor)
				}
			} else {
				fmt.Print(cursorPadding)
			}
			fmt.Println(consoleCommand[i])
		}
		control := Scan('\n')
		if control == "" {
			if selectFriendCursor {
				if selectedFriend[friendCursorIndex] == false {
					selectedFriendNum++
					selectedFriend[friendCursorIndex] = true
				}
			} else {
				if consoleCursorIndex == 0 {
					flush := true
					list := []GroupChatRecord{}
					listTmp := []string{}
					cursorIndex := 0
					cursorLen := 0
					for {
						if flush {
							chatNode.groupsLock.RLock()
							list = list[:0]
							listTmp = listTmp[:0]
							for name, groups := range chatNode.groups {
								for _, group := range groups {
									list = append(list, group)
									listTmp = append(listTmp, name)
								}
							}
							chatNode.groupsLock.RUnlock()
							cursorIndex = 0
							cursorLen = len(list)
							flush = false
						}
						fmt.Println(flushPadding)
						PrintCentre("Select the group chat you will invite your friends in.", "yellow")
						if cursorLen == 0 {
							PrintCentre("You haven't been in any chat group yet.", "red")
						} else {
							for i := 0; i < cursorLen; i++ {
								if i == cursorIndex {
									blue.Print(cursor)
								} else {
									fmt.Print(cursorPadding)
								}
								yellow.Print(listTmp[i])
								fmt.Println(" Started at", list[i].GroupStartTime.Format("2006-01-02 15:04"))
							}
						}
						fmt.Println(separator)
						PrintCentre("Press up/down arrow to move the cursor.", "magenta")
						PrintCentre("Press enter to confirm.", "magenta")
						PrintCentre("Press left arrow to return to superior page.", "magenta")
						PrintCentre("Press right arrow to create a new chat group.", "magenta")
						control := Scan('\n')
						if control == leftArrow {
							break
						} else if control == rightArrow {
							fmt.Println(flushPadding)
							PrintCentre("Please type new group chat name:", "yellow")
							name := Scan('\n')
							chatNode.CreateChatGroup(name)
							flush = true
							fmt.Println(flushPadding)
							PrintCentre("Successfully create the chat group.", "green")
							PrintCentre("Type anything to continue.", "white")
							Scan('\n')
						} else if control == "" {
							fmt.Println(flushPadding)
							if selectedFriendNum == 0 {
								PrintCentre("You have not selecte any friend yet!", "red")
							} else {
								PrintCentre("Sending invitation to "+strconv.Itoa(selectedFriendNum)+" friends", "yellow")
								for i, selected := range selectedFriend {
									if selected {
										err := chatNode.InviteFriend(friendList[i], listTmp[cursorIndex], list[cursorIndex])
										if err != nil {
											PrintCentre(friendList[i]+": "+err.Error(), "red")
										} else {
											PrintCentre("Successfully send invitation to "+friendList[i], "green")
										}
									}
								}
							}
							PrintCentre("Type anything to continue.", "white")
							Scan('\n')
						} else {
							cursorIndex = moveCursor(cursorLen, cursorIndex, control)
						}
					}
					continue
				} else if consoleCursorIndex == 1 {
					return "homePage"
				} else if consoleCursorIndex == 2 {
					return "help"
				}
			}
		}
		if control == leftArrow {
			selectFriendCursor = true
		} else if control == rightArrow {
			selectFriendCursor = false
		} else if control == string('\t') {
			for i := range selectedFriend {
				selectedFriend[i] = false
			}
			selectedFriendNum = 0
		} else {
			if selectFriendCursor {
				friendCursorIndex = moveCursor(friendCursorLen, friendCursorIndex, control)
			} else {
				consoleCursorIndex = moveCursor(3, consoleCursorIndex, control)
			}
		}
	}
}

// 查看群聊列表
func (chatNode *ChatNode) viewGroupChatList() string {
	flush := true
	list := []GroupChatRecord{}
	listTmp := []string{}
	cursorIndex := 0
	cursorLen := 0
	consoleCursorIndex := 0
	selectGroup := false
	selectGroupIndex := -1
	consoleCommand := []string{"Enter the group chat", "Create a new group chat", "Back to home page", "help"}
	for {
		if flush {
			chatNode.groupsLock.RLock()
			list = list[:0]
			listTmp = listTmp[:0]
			for name, groups := range chatNode.groups {
				for _, group := range groups {
					list = append(list, group)
					listTmp = append(listTmp, name)
				}
			}
			chatNode.groupsLock.RUnlock()
			cursorIndex = 0
			cursorLen = len(list)
			consoleCursorIndex = 0
			selectGroup = false
			selectGroupIndex = -1
			flush = false
		}
		fmt.Println(flushPadding)
		PrintCentre("Group chat list:", "yellow")
		if cursorLen == 0 {
			PrintCentre("You haven't been in any chat group yet!", "red")
		} else {
			for i := 0; i < cursorLen; i++ {
				if i == cursorIndex {
					if selectGroup {
						blue.Print(cursor)
					} else {
						fmt.Print(cursor)
					}
				} else {
					fmt.Print(cursorPadding)
				}
				if i == selectGroupIndex {
					green.Print(listTmp[i])
				} else {
					yellow.Print(listTmp[i])
				}
				fmt.Println(" Started at", list[i].GroupStartTime.Format("2006-01-02 15:04"))
			}
		}
		fmt.Println(separator)
		PrintCentre("Press up/down arrow to move the cursor.", "magenta")
		PrintCentre("Press left/right arrow to get the groupList/console cursor.", "magenta")
		PrintCentre("Press enter to confirm.", "magenta")
		for i := 0; i < 4; i++ {
			if i == consoleCursorIndex {
				if selectGroup {
					fmt.Print(cursor)
				} else {
					blue.Print(cursor)
				}
			} else {
				fmt.Print(cursorPadding)
			}
			fmt.Println(consoleCommand[i])
		}
		control := Scan('\n')
		if control == leftArrow {
			selectGroup = true
		} else if control == rightArrow {
			selectGroup = false
		} else if control == "" {
			if selectGroup {
				selectGroupIndex = cursorIndex
			} else {
				if consoleCursorIndex == 0 {
					if cursorLen == 0 || selectGroupIndex == -1 {
						fmt.Println(flushPadding)
						PrintCentre("You haven't select any chat group yet!", "red")
						PrintCentre("Type anything to continue.", "white")
						Scan('\n')
					} else {
						beginTime := time.Now()
						moreHistoricalMessage := true
						for {
							fmt.Println(flushPadding)
							PrintCentre("Group Chat: "+listTmp[selectGroupIndex], "yellow")
							if moreHistoricalMessage {
								PrintCentre("^^^", "blue")
								PrintCentre("More historical messages", "blue")
							} else {
								PrintCentre("No more historical messages", "red")
							}
							infos, err := chatNode.GetChatInfo(list[selectGroupIndex], beginTime, time.Now())
							if err != nil {
								PrintCentre(err.Error(), "red")
							} else {
								chatNode.printChatRecords(infos)
							}
							fmt.Println(separator)
							PrintCentre("Press enter to refresh the chat records.", "magenta")
							PrintCentre("Press up arrow to get more historical messages.", "magenta")
							PrintCentre("Press left arrow to return to superior page.", "magenta")
							PrintCentre("Type messages to chat", "magenta")
							control := Scan('\n')
							if control == "" {
								continue
							} else if control == leftArrow {
								flush = true
								break
							} else if control == upArrow {
								earlierTime, err := chatNode.GetEarlierChatInfoTime(list[selectGroupIndex], beginTime)
								if err == nil {
									beginTime = earlierTime
								} else {
									moreHistoricalMessage = false
								}
							} else {
								err = chatNode.SendChatInfo(control, list[selectGroupIndex])
								if err != nil {
									PrintCentre(err.Error(), "red")
									PrintCentre("Type anything to continue.", "white")
									Scan('\n')
								}
							}
						}
					}
				} else if consoleCursorIndex == 1 {
					fmt.Println(flushPadding)
					PrintCentre("Please type new group chat name:", "yellow")
					name := Scan('\n')
					chatNode.CreateChatGroup(name)
					flush = true
					fmt.Println(flushPadding)
					PrintCentre("Successfully create the chat group.", "green")
					PrintCentre("Type anything to continue.", "white")
					Scan('\n')
				} else if consoleCursorIndex == 2 {
					return "homePage"
				} else if consoleCursorIndex == 3 {
					return "help"
				}
			}
		} else {
			if selectGroup {
				cursorIndex = moveCursor(cursorLen, cursorIndex, control)
			} else {
				consoleCursorIndex = moveCursor(4, consoleCursorIndex, control)
			}
		}
	}
}

// 查看好友请求
func (chatNode *ChatNode) viewFriendRequest() string {
	cursorIndex := 0
	cursorLen := 0
	consoleCursorIndex := 0
	status := 0 //0表初始状态，1表待确认列表，2表已发送列表
	selectRequest := false
	list := []string{}
	listTmp := []string{}
	consoleCommand := []string{"Back to home page", "help"}
	for {
		fmt.Println(flushPadding)
		if status == 0 {
			chatNode.friendRequestLock.RLock()
			friendRequestLen := len(chatNode.friendRequest)
			chatNode.friendRequestLock.RUnlock()
			chatNode.sentFriendRequestLock.RLock()
			sentFriendRequestLen := len(chatNode.sentFriendRequest)
			chatNode.sentFriendRequestLock.RUnlock()
			PrintCentre(strconv.Itoa(friendRequestLen)+" requests to confirm.", "yellow")
			PrintCentre(strconv.Itoa(sentFriendRequestLen)+" requests were sent.", "yellow")
		} else if status == 1 {
			if cursorLen == 0 {
				PrintCentre("No friend request!", "red")
			} else {
				PrintCentre("Friend requests to confirm:", "yellow")
				for i, name := range list {
					if i == cursorIndex {
						if selectRequest {
							blue.Print(cursor)
						} else {
							fmt.Print(cursor)
						}
					} else {
						fmt.Print(cursorPadding)
					}
					yellow.Println(name)
				}
			}
		} else if status == 2 {
			if cursorLen == 0 {
				PrintCentre("No request was sent!", "red")
			} else {
				PrintCentre("Friend requests have sent:", "yellow")
				for i, name := range list {
					if i == cursorIndex {
						if selectRequest {
							blue.Print(cursor)
						} else {
							fmt.Print(cursor)
						}
					} else {
						fmt.Print(cursorPadding)
					}
					yellow.Print(name)
					fmt.Print(" --- ")
					if listTmp[i] == "Accepted" {
						green.Println("Accepted")
					} else if listTmp[i] == "Rejected" {
						red.Println("Rejected")
					} else {
						yellow.Println("To be confirmed")
					}
				}
			}
		}
		fmt.Println(separator)
		PrintCentre("Press up/down arrow to move the cursor.", "magenta")
		PrintCentre("Press left/right arrow to get the friendRequest/console cursor.", "magenta")
		PrintCentre("Press enter to confirm.", "magenta")
		PrintCentre("Press Tab to shift between (requests to confirm / requests have sent).", "magenta")
		for i := 0; i < 2; i++ {
			if consoleCursorIndex == i {
				if selectRequest {
					fmt.Print(cursor)
				} else {
					blue.Print(cursor)
				}
			} else {
				fmt.Print(cursorPadding)
			}
			fmt.Println(consoleCommand[i])
		}
		control := Scan('\n')
		if control == leftArrow {
			selectRequest = true
		} else if control == rightArrow {
			selectRequest = false
		} else if control == "" || control == string('\t') {
			if control == "" {
				if selectRequest {
					if status == 1 && cursorLen > 0 {
						fmt.Println(flushPadding)
						PrintCentre("Do you agree? Please type Y(y)/N(n):", "yellow")
						agree := Scan('\n')
						err := chatNode.CheckFriendRequest(list[cursorIndex], (agree == "Y" || agree == "y"))
						fmt.Println(flushPadding)
						if err != nil {
							PrintCentre(err.Error(), "red")
						} else {
							PrintCentre("Successfully accept the request.", "green")
						}
						PrintCentre("Type anything to continue.", "white")
						Scan('\n')
					} else {
						if status == 2 && cursorLen > 0 {
							fmt.Println(flushPadding)
							PrintCentre("Delete the accepted/rejected record? Please type Y(y)/N(n):", "yellow")
							agree := Scan('\n')
							if agree == "Y" || agree == "y" {
								err := chatNode.DeleteSentRequest(list[cursorIndex])
								fmt.Println(flushPadding)
								if err != nil {
									PrintCentre(err.Error(), "red")
								} else {
									PrintCentre("Successfully accept the request.", "green")
								}
								PrintCentre("Type anything to continue.", "white")
								Scan('\n')
							}
						}
					}
				} else {
					if consoleCursorIndex == 0 {
						return "homePage"
					} else if consoleCursorIndex == 1 {
						return "help"
					}
				}
			}
			if !(control == "" && !selectRequest) {
				if (control == "" && status == 1) ||
					(control == string('\t') && (status == 0 || status == 2)) {
					chatNode.friendRequestLock.RLock()
					list = list[:0]
					for name := range chatNode.friendRequest {
						list = append(list, name)
					}
					chatNode.friendRequestLock.RUnlock()
					cursorIndex = 0
					cursorLen = len(list)
					consoleCursorIndex = 0
					selectRequest = false
					status = 1
				} else if (control == "" && status == 2) || (control == string('\t') && status == 1) {
					chatNode.sentFriendRequestLock.RLock()
					list = list[:0]
					listTmp = listTmp[:0]
					for name, value := range chatNode.sentFriendRequest {
						list = append(list, name)
						if value == "Accepted" || value == "Rejected" {
							listTmp = append(listTmp, value)
						} else {
							listTmp = append(listTmp, "To be confirmed")
						}
					}
					chatNode.sentFriendRequestLock.RUnlock()
					cursorIndex = 0
					cursorLen = len(list)
					consoleCursorIndex = 0
					selectRequest = false
					status = 2
				}
			}
		} else {
			if selectRequest {
				cursorIndex = moveCursor(cursorLen, cursorIndex, control)
			} else {
				consoleCursorIndex = moveCursor(2, consoleCursorIndex, control)
			}
		}
	}
}

// 查看群聊邀请
func (chatNode *ChatNode) viewGroupChatInvitation() string {
	list := []InvitationPair{}
	listTmp := []string{}
	flush := true
	cursorIndex := 0
	cursorLen := 0
	for {
		if flush {
			list = list[:0]
			listTmp = listTmp[:0]
			chatNode.invitationLock.RLock()
			for name, invitations := range chatNode.invitation {
				for _, invitation := range invitations {
					list = append(list, invitation)
					listTmp = append(listTmp, name)
				}
			}
			chatNode.invitationLock.RUnlock()
			cursorIndex = 0
			cursorLen = len(list)
			flush = false
		}
		fmt.Println(flushPadding)
		if cursorLen == 0 {
			PrintCentre("You haven't received any chat group invitation yet!", "red")
		} else {
			PrintCentre("You have "+strconv.Itoa(cursorLen)+" invitations to confirm.", "yellow")
			for i := 0; i < cursorLen; i++ {
				if i == cursorIndex {
					blue.Print(cursor)
				} else {
					fmt.Print(cursorPadding)
				}
				yellow.Print(listTmp[i])
				fmt.Println(" From", list[i].FromName, "Started at", list[i].GroupStartTime.Format("2006-01-02 15:04"))
			}
		}
		fmt.Println(separator)
		PrintCentre("Press up/down arrow to move the cursor.", "magenta")
		PrintCentre("Press enter to confirm.", "magenta")
		PrintCentre("Press left arrow to return to superior page.", "magenta")
		control := Scan('\n')
		if control == leftArrow {
			return "homePage"
		} else if control == "" {
			if cursorLen == 0 {
				fmt.Println(flushPadding)
				PrintCentre("You haven't received any chat group invitation yet!", "red")
			} else {
				fmt.Println(flushPadding)
				PrintCentre("Do you agree? Please type Y(y)/N(n):", "yellow")
				agree := Scan('\n')
				err := chatNode.CheckInvitation(list[cursorIndex], (agree == "Y" || agree == "y"))
				fmt.Println(flushPadding)
				if err != nil {
					PrintCentre(err.Error(), "red")
				} else {
					PrintCentre("Successfully enter the group ", "green")
				}
				flush = true
			}
			PrintCentre("Type anything to continue.", "white")
			Scan('\n')
		}
	}
}

// 添加好友
func (chatNode *ChatNode) addNewFriend() string {
	for {
		fmt.Println(flushPadding)
		PrintCentre("Add New friend!", "yellow")
		fmt.Println(separator)
		PrintCentre("Type the friend Name.", "magenta")
		PrintCentre("Press enter to return to the home page.", "magenta")
		name := Scan('\n')
		if name == "" {
			return "homePage"
		}
		err := chatNode.SendFriendRequest(name)
		fmt.Println(flushPadding)
		if err != nil {
			PrintCentre(err.Error(), "red")
		} else {
			PrintCentre("Request was sent successfully.", "green")
		}
		PrintCentre("Type anything to continue.", "white")
		Scan('\n')
	}
}

// 帮助
func (chatNode *ChatNode) help() string {
	fmt.Println(flushPadding)
	PrintCentre("To be developed", "red")
	fmt.Println(separator)
	PrintCentre("Type anything to return home page.", "magenta")
	Scan('\n')
	return "homePage"
}

// 退出
func (chatNode *ChatNode) exit() {
	fmt.Println(flushPadding)
	PrintCentre("Goodbye!", "yellow")
}
