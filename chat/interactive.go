package chat

import (
	"fmt"
	"strconv"
	"time"
)

// 主程序
func Chat() {
	var err error
	node := new(ChatNode)
	cursorIndex := 0
	consoleCommand := []string{"Login to an existing account", "Register a new account", "Help", "About"}
	userName := ""
	userIp := ""
	password := ""
	enterIp := ""
	create := false
	for {
		fmt.Println(flushPadding)
		printLogo()
		PrintCentre("Welcome to WGJ's P2P chat!", "yellow")
		printConsole(4, cursorIndex, consoleCommand)
		fmt.Println(separator)
		PrintCentre("Press up/down arrow to move the cursor.", "magenta")
		PrintCentre("Press enter to confirm.", "magenta")
		control := Scan('\n')
		if control == "" {
			if cursorIndex == 0 || cursorIndex == 1 {
				PrintCentre("Please type your name:", "yellow")
				userName = Scan('\n')
				PrintCentre("Please type your IP:", "yellow")
				userIp = Scan('\n')
				PrintCentre("Please type your password:", "yellow")
				password = Scan('\n')
				if cursorIndex == 1 {
					PrintCentre("Do you want to create a new P2P net?", "yellow")
					PrintCentre("Type Y(y)/N(n):", "yellow")
					create = getSelection()
				}
				if !create {
					PrintCentre("Please type the node IP (the online node is in the P2P chat net):", "yellow")
					enterIp = Scan('\n')
				}
				err = node.Login(userName, userIp, password, enterIp, cursorIndex == 1)
				if err != nil {
					PrintCentre(err.Error(), "red")
				} else {
					PrintCentre("Successfully log in.", "green")
				}
				break
			} else if cursorIndex == 2 {
				node.help()
			} else if cursorIndex == 3 {
				fmt.Println(flushPadding)
				yellow.Println("This chat program is developed by WGJ during PPCA 2023.")
				yellow.Println("The chat program is an interesting application of distributed hash table.")
				yellow.Println("This chat program is based on chord protocol.")
				fmt.Println("Type anything to continue.")
				Scan('\n')
			}
		} else {
			cursorIndex = moveCursor(4, cursorIndex, control)
		}
	}
	PrintCentre("Type anything to continue.", "white")
	Scan('\n')
	if err == nil {
		node.interactive()
		node.LogOut()
	} else {
		node.help()
	}
}

// 交互页面间的状态转移
func (chatNode *ChatNode) interactive() {
	next := "homepage"
	for {
		switch next {
		case "homepage":
			next = chatNode.homepage()
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
		case "exit":
			chatNode.exit()
			return
		}
	}
}

// 首页
func (chatNode *ChatNode) homepage() string {
	cursorIndex := 0
	consoleCommand := []string{"View friend list", "View group chat list",
		"View friend request", "View group chat invitation", "Add new friend", "Help", "Exit"}
	for {
		fmt.Println(flushPadding)
		PrintCentre(chatNode.name+", welcome to WGJ's P2P chat!", "yellow")
		fmt.Println(separator)
		PrintCentre("Press up/down arrow to move the cursor.", "magenta")
		PrintCentre("Press enter to confirm.", "magenta")
		printConsole(7, cursorIndex, consoleCommand)
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
				chatNode.help()
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
	friendList, privateChatList := chatNode.GetFriendList()
	friendCursorIndex := 0
	friendCursorLen := len(friendList)
	consoleCursorIndex := 0
	selectedFriend := make([]bool, friendCursorLen)
	selectFriendCursor := false
	selectedFriendNum := 0
	consoleCommand := []string{"Invite selected friends to group chat",
		"Chat privately with selected friends", "Back to homepage", "Help"}
	for {
		fmt.Println(flushPadding)
		yellow.Println("Friend list:")
		if len(friendList) == 0 {
			PrintCentre("You don't have any friends now!", "red")
		} else {
			for i, name := range friendList {
				if i == friendCursorIndex {
					if selectFriendCursor {
						cyan.Print(cursor)
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
		printConsoleSelected(4, consoleCursorIndex, consoleCommand, selectFriendCursor)
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
									cyan.Print(cursor)
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
							} else if cursorLen == 0 {
								PrintCentre("You haven't been in any chat group yet.", "red")
							} else {
								PrintCentre("Sending invitation to "+strconv.Itoa(selectedFriendNum)+" friends", "yellow")
								for i, selected := range selectedFriend {
									if selected {
										err := chatNode.InviteFriend(friendList[i], listTmp[cursorIndex], list[cursorIndex])
										if err != nil {
											if err.Error() == "Pending." {
												PrintCentre("The friend "+friendList[i]+" is not online. The invitation will be sent automatically after he/she login.", "blue")
											} else {
												PrintCentre("Fail to send invitation to "+friendList[i]+": "+err.Error(), "red")
											}
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
					for i := range selectedFriend {
						selectedFriend[i] = false
					}
					selectedFriendNum = 0
					friendCursorIndex = 0
					consoleCursorIndex = 0
					continue
				} else if consoleCursorIndex == 1 {
					if selectedFriendNum != 1 {
						fmt.Println(flushPadding)
						if selectedFriendNum < 1 {
							PrintCentre("You haven't selected any friend yet!", "red")
						} else {
							PrintCentre("You have selected more than one friend!", "red")
						}
						PrintCentre("Type anything to continue.", "white")
						Scan('\n')
						continue
					}
					friendIndex := 0
					for i := range selectedFriend {
						if selectedFriend[i] {
							friendIndex = i
							break
						}
					}
					beginTime := time.Now()
					moreHistoricalMessage := true
					for {
						fmt.Println(flushPadding)
						PrintCentre("Private Chat with: "+friendList[friendIndex], "yellow")
						if moreHistoricalMessage {
							PrintCentre("^^^", "cyan")
							PrintCentre("More historical messages", "cyan")
						} else {
							PrintCentre("No more historical messages", "red")
						}
						infos, err := chatNode.GetChatInfo(privateChatList[friendIndex], beginTime, time.Now())
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
							break
						} else if control == upArrow {
							earlierTime, err := chatNode.GetEarlierChatInfoTime(privateChatList[friendIndex], beginTime)
							if err == nil {
								beginTime = earlierTime
							} else {
								moreHistoricalMessage = false
							}
						} else {
							err = chatNode.SendChatInfo(control, privateChatList[friendIndex])
							if err != nil {
								PrintCentre(err.Error(), "red")
								PrintCentre("Type anything to continue.", "white")
								Scan('\n')
							}
						}
					}
					for i := range selectedFriend {
						selectedFriend[i] = false
					}
					selectedFriendNum = 0
					friendCursorIndex = 0
					consoleCursorIndex = 0
					continue
				} else if consoleCursorIndex == 2 {
					return "homepage"
				} else if consoleCursorIndex == 3 {
					chatNode.help()
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
				consoleCursorIndex = moveCursor(4, consoleCursorIndex, control)
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
	consoleCommand := []string{"Enter the group chat", "Create a new group chat", "Back to homepage", "Help"}
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
						cyan.Print(cursor)
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
					fmt.Println(cursor + consoleCommand[i])
				} else {
					cyan.Println(cursor + consoleCommand[i])
				}
			} else {
				fmt.Println(cursorPadding + consoleCommand[i])
			}
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
								PrintCentre("^^^", "cyan")
								PrintCentre("More historical messages", "cyan")
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
					return "homepage"
				} else if consoleCursorIndex == 3 {
					chatNode.help()
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
	list := []string{}
	listTmp := []string{}
	consoleCommand := []string{"Requests to confirm", "Requests have sent", "Back to homepage", "Help"}
	for {
		fmt.Println(flushPadding)

		chatNode.friendRequestLock.RLock()
		friendRequestLen := len(chatNode.friendRequest)
		chatNode.friendRequestLock.RUnlock()
		chatNode.sentFriendRequestLock.RLock()
		sentFriendRequestLen := len(chatNode.sentFriendRequest)
		chatNode.sentFriendRequestLock.RUnlock()
		PrintCentre(strconv.Itoa(friendRequestLen)+" requests to confirm.", "yellow")
		PrintCentre(strconv.Itoa(sentFriendRequestLen)+" requests were sent.", "yellow")
		fmt.Println(separator)
		PrintCentre("Press up/down arrow to move the cursor.", "magenta")
		PrintCentre("Press enter to confirm.", "magenta")
		printConsole(4, consoleCursorIndex, consoleCommand)

		control := Scan('\n')
		if control == "" {
			if consoleCursorIndex == 0 {
				flush := true
				for {
					if flush {
						chatNode.friendRequestLock.RLock()
						list = list[:0]
						for name, sent := range chatNode.friendRequest {
							if sent.ChatSeed == "" {
								list = append(list, name)
							}
						}
						chatNode.friendRequestLock.RUnlock()
						cursorIndex = 0
						cursorLen = len(list)
						consoleCursorIndex = 0
						flush = false
					}
					fmt.Println(flushPadding)
					if cursorLen == 0 {
						PrintCentre("No friend request!", "red")
					} else {
						PrintCentre("Friend requests to confirm:", "yellow")
						for i, name := range list {
							if i == cursorIndex {
								cyan.Println(cursor + name)
							} else {
								fmt.Println(cursorPadding + name)
							}
						}
					}
					fmt.Println(separator)
					PrintCentre("Press up/down arrow to move the cursor.", "magenta")
					PrintCentre("Press left arrow to return to superior page.", "magenta")
					PrintCentre("Press enter to select the request you want to confirm.", "magenta")
					control := Scan('\n')
					if control == leftArrow {
						break
					} else if control == "" {
						fmt.Println(flushPadding)
						if cursorLen == 0 {
							PrintCentre("No friend request!", "red")
							PrintCentre("Type anything to continue.", "white")
							Scan('\n')
							continue
						}
						PrintCentre("Do you agree? Please type Y(y)/N(n):", "yellow")
						agree := getSelection()
						err := chatNode.CheckFriendRequest(list[cursorIndex], agree)
						fmt.Println(flushPadding)
						if err != nil {
							PrintCentre(err.Error(), "red")
						} else if agree {
							PrintCentre("Successfully accept the request.", "green")
						} else {
							PrintCentre("Rejected the request.", "red")
						}
						flush = true
						PrintCentre("Type anything to continue.", "white")
						Scan('\n')
					} else {
						cursorIndex = moveCursor(cursorLen, cursorIndex, control)
					}
				}
			} else if consoleCursorIndex == 1 {
				flush := true
				for {
					if flush {
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
						flush = false
					}
					fmt.Println(flushPadding)
					if cursorLen == 0 {
						PrintCentre("No request was sent!", "red")
					} else {
						PrintCentre("Friend requests have sent:", "yellow")
						for i, name := range list {
							if i == cursorIndex {
								cyan.Print(cursor)
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
					fmt.Println(separator)
					PrintCentre("Press up/down arrow to move the cursor.", "magenta")
					PrintCentre("Press left arrow to return to superior page.", "magenta")
					PrintCentre("Press enter to select the request record you want to delete.", "magenta")

					control := Scan('\n')

					if control == leftArrow {
						break
					} else if control == "" {
						fmt.Println(flushPadding)
						if cursorLen == 0 {
							PrintCentre("No request was sent!", "red")
							PrintCentre("Type anything to continue.", "white")
							Scan('\n')
							continue
						}
						PrintCentre("Delete the accepted/rejected record?", "yellow")
						PrintCentre("Please type Y(y)/N(n):", "yellow")
						agree := getSelection()
						if agree {
							err := chatNode.DeleteSentRequest(list[cursorIndex])
							fmt.Println(flushPadding)
							if err != nil {
								PrintCentre(err.Error(), "red")
							} else {
								PrintCentre("Successfully accept the request.", "green")
							}
							flush = true
							PrintCentre("Type anything to continue.", "white")
							Scan('\n')
						}
					} else {
						cursorIndex = moveCursor(cursorLen, cursorIndex, control)
					}

				}

			} else if consoleCursorIndex == 2 {
				return "homepage"
			} else if consoleCursorIndex == 3 {
				chatNode.help()
			}
		} else {
			consoleCursorIndex = moveCursor(4, consoleCursorIndex, control)
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
					cyan.Print(cursor)
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
			return "homepage"
		} else if control == "" {
			if cursorLen == 0 {
				fmt.Println(flushPadding)
				PrintCentre("You haven't received any chat group invitation yet!", "red")
			} else {
				fmt.Println(flushPadding)
				PrintCentre("Do you agree? Please type Y(y)/N(n):", "yellow")
				agree := getSelection()
				err := chatNode.CheckInvitation(list[cursorIndex], agree)
				fmt.Println(flushPadding)
				if err != nil {
					PrintCentre(err.Error(), "red")
				} else {
					if agree {
						PrintCentre("Successfully enter the group.", "green")
					} else {
						PrintCentre("Reject to enter the group.", "red")
					}
				}
				flush = true
			}
			PrintCentre("Type anything to continue.", "white")
			Scan('\n')
		} else {
			cursorIndex = moveCursor(cursorLen, cursorIndex, control)
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
		PrintCentre("Press enter to return to the homepage.", "magenta")
		name := Scan('\n')
		if name == "" {
			return "homepage"
		}
		err := chatNode.SendFriendRequest(name)
		fmt.Println(flushPadding)
		if err != nil {
			if err.Error() == "Pending." {
				PrintCentre("The user "+name+" is not online. The request will be sent automatically after he/she login.", "blue")
			} else {
				PrintCentre(err.Error(), "red")
			}
		} else {
			PrintCentre("Request was sent to "+name+" successfully.", "green")
		}
		PrintCentre("Type anything to continue.", "white")
		Scan('\n')
	}
}

// 帮助
func (chatNode *ChatNode) help() string {
	cursorIndex := 0
	consoleCommand := []string{"How to login?", "How to add new friend?",
		"How to create a new chat group?", "How to invite my friends to chat group?",
		"How to chat privately with my friend?",
		"Return to superior page"}
	for {
		fmt.Println(flushPadding)
		PrintCentre("How can I help you?", "yellow")
		fmt.Println(separator)
		PrintCentre("Press up/down arrow to move the cursor.", "magenta")
		PrintCentre("Press enter to confirm.", "magenta")
		printConsole(6, cursorIndex, consoleCommand)
		control := Scan('\n')
		if control == "" {
			if cursorIndex == 0 {
				fmt.Println(flushPadding)
				yellow.Print("If you want to login the registed account")
				green.Print("(Login to an existing account)")
				yellow.Println(", you need type in account's name, your current IP and account's password correctly first. " +
					"Then you need to type in the IP of a node that is currently online in that system.")
				yellow.Print("If you don't have an account, you can register an account first")
				green.Print("(Register a new account)")
				yellow.Println(". Type in new account's name, your current IP and new account's password. " +
					"If you want the account to join in an existed P2P system, you need to type in the IP of a node that is currently online in that system. " +
					"If you just want to create a new P2P system, you needn't do that.")
				blue.Println("* Please ensure that the IP you type in is correct, otherwise unknown situations may occur.")
				blue.Println("* After creating the P2P system, please ensure that at least one server is always online in the system, otherwise all data will be lost.")
				red.Println("! The user name cannot be repeated. If you type in a name that is existed, you will get error message and you should type in a new name.")
				red.Println("! If you type in the password incorrectly, you cannot login successfully. You need type again.")
				red.Println("! An account cannot login to multiple devices simultaneously.")
			} else if cursorIndex == 1 {
				fmt.Println(flushPadding)
				yellow.Print("To add a new friend, you should go to ")
				green.Print("homepage -> Add new friend")
				yellow.Print(".\nType in the name of the user you want to add as your friend. The requset will be sent to the user, and you can view its status in ")
				green.Print("homepage -> View friend request -> Requests have sent")
				yellow.Println(".")
				red.Println("! You cannot send request to the user which is not existed.")
				red.Println("! You cannot send request to yourself.")
				red.Println("! You cannot send request to your friend.")
				red.Println("! You cannot send request to the user which you have sent request already. You should wait that user to confirm your request.")
				red.Print("! You cannot send request to the user which you have already received the request from that user. You should go to ")
				magenta.Print("homepage -> View friend request -> Requests to confirm")
				red.Println(" to check your friend request list.")
			} else if cursorIndex == 2 {
				fmt.Println(flushPadding)
				yellow.Println("You can create a new chat group in two ways:")
				yellow.Print("1. Go to ")
				green.Print("homepage -> View group chat list -> Create a new group chat")
				yellow.Println(", than type in the new chat group name.")
				yellow.Print("2. Go to ")
				green.Print("homepage -> View friend list -> Invite selected friends to group chat")
				yellow.Println(", than press right arrow, typing in the new chat group name.")
				blue.Println("* The name of chat group can be repeated.")
			} else if cursorIndex == 3 {
				fmt.Println(flushPadding)
				yellow.Print("To invite friends to chat group, you should go to ")
				green.Print("homepage -> View friend list")
				yellow.Println(", than pressing left arrow to shift cursor to friendList area.")
				yellow.Println("You can select the friends you want to invite by moving cursor on his/her name and press enter. " +
					"The name of selected friends will turn to green. You can clear your selection by pressing tab.")
				yellow.Print("After selection, press right arrow to shift cursor to console area. Go to ")
				green.Print("Invite selected friends to group chat")
				yellow.Println(". Move the cursor to the name of the group chat which you want to invite your friends in, than pressing enter to confirm. " +
					"If you haven't had any chat group yet, ypu can also press right arrow to create a new one.")
				yellow.Println("Then the console will display the invitation sending status.")
				blue.Println("* The friend you send invitation to will enter the chat group after he/she confirm the invitation.")
				red.Println("! If you don't select any friends, you will get error message, and you need to select again.")
				red.Println("! If you don't have any chat group and try to press enter, you will get error message. You need to creat a new group first.")
				red.Println("! You cannot send invitation to the friend which has been in the group already.")
				red.Println("! You cannot send invitation to the friend which has been reveived the invitation to the group. You should wait him/her to confirm.")
			} else if cursorIndex == 4 {
				fmt.Println(flushPadding)
				yellow.Print("To chat with friends privately, you should go to ")
				green.Print("homepage -> View friend list")
				yellow.Print(". Move the cursor and press enter to select the friend you want to chat with. Then go to ")
				green.Print("(homepage -> View friend list) -> Chat privately with selected friends")
				yellow.Println(". Now you can chat with your friend privately.")
				blue.Println("* You can get more historical messages of the chat by pressing uparrow.")
			} else if cursorIndex == 5 {
				return "homepage"
			}
			fmt.Println("Type anything to return to superior page.")
			Scan('\n')
		} else {
			cursorIndex = moveCursor(6, cursorIndex, control)
		}
	}
}

// 退出
func (chatNode *ChatNode) exit() {
	fmt.Println(flushPadding)
	PrintCentre("Goodbye!", "yellow")
}
