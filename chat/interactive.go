package chat

import (
	"fmt"
	"strconv"
	"time"
)

func (chatNode *ChatNode) Interactive() {
	next := "HomePage"
	for {
		switch next {
		case "HomePage":
			next = chatNode.HomePage()
		case "ViewFriendList":
			next = chatNode.ViewFriendList()
		case "ViewFriendRequest":
			next = chatNode.ViewFriendRequest()
		case "AddNewFriend":
			next = chatNode.AddNewFriend()
		case "ViewGroupChatInvitation":
			next = chatNode.ViewGroupChatInvitation()
		case "ViewGroupChatList":
			next = chatNode.ViewGroupChatList()
		case "Help":
			next = chatNode.Help()
		case "Quit":
			chatNode.Quit()
			return
		}
	}
}

func (chatNode *ChatNode) HomePage() string {
	cursorIndex := 0
	consoleCommand := []string{"View friend list", "View group chat list", "View friend request", "View group chat invitation", "Add new friend", "Help", "Quit"}
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
				return "ViewFriendList"
			} else if cursorIndex == 1 {
				return "ViewGroupChatList"
			} else if cursorIndex == 2 {
				return "ViewFriendRequest"
			} else if cursorIndex == 3 {
				return "ViewGroupChatInvitation"
			} else if cursorIndex == 4 {
				return "AddNewFriend"
			} else if cursorIndex == 5 {
				return "Help"
			} else if cursorIndex == 6 {
				return "Quit"
			}
		} else {
			cursorIndex = moveCursor(7, cursorIndex, control)
		}
	}
}

func (chatNode *ChatNode) ViewFriendList() string {
	friendList := chatNode.GetFriendList()
	friendCursorIndex := 0
	friendCursorLen := len(friendList)
	consoleCursorIndex := 0
	selectedFriend := make([]bool, friendCursorLen)
	selectFriendCursor := false
	selectedFriendNum := 0
	consoleCommand := []string{"Invite selected friends to group chat", "Back to home page", "Help"}
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
					return "HomePage"
				} else if consoleCursorIndex == 2 {
					return "Help"
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

func (chatNode *ChatNode) ViewFriendRequest() string {
	cursorIndex := 0
	cursorLen := 0
	consoleCursorIndex := 0
	status := 0 //0表初始状态，1表待确认列表，2表已发送列表
	selectRequest := false
	list := []string{}
	listTmp := []string{}
	consoleCommand := []string{"Back to home page", "Help"}
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
						return "HomePage"
					} else if consoleCursorIndex == 1 {
						return "Help"
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

func (chatNode *ChatNode) AddNewFriend() string {
	for {
		fmt.Println(flushPadding)
		PrintCentre("Add New friend!", "yellow")
		fmt.Println(separator)
		PrintCentre("Type the friend Name.", "magenta")
		PrintCentre("Press enter to return to the home page.", "magenta")
		name := Scan('\n')
		if name == "" {
			return "HomePage"
		}
		err := chatNode.AddFriend(name)
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

func (chatNode *ChatNode) ViewGroupChatInvitation() string {
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
			return "HomePage"
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
					PrintCentre("Successfully enter the group chat.", "green")
				}
				flush = true
			}
			PrintCentre("Type anything to continue.", "white")
			Scan('\n')
		}
	}
}

func (chatNode *ChatNode) ViewGroupChatList() string {
	flush := true
	list := []GroupChatRecord{}
	listTmp := []string{}
	cursorIndex := 0
	cursorLen := 0
	consoleCursorIndex := 0
	selectGroup := false
	selectGroupIndex := -1
	consoleCommand := []string{"Enter the group chat", "Create a new group chat", "Back to home page", "Help"}
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
						for {
							fmt.Println(flushPadding)
							PrintCentre(listTmp[selectGroupIndex], "yellow")
							infos, err := chatNode.GetChatInfo(list[selectGroupIndex], time.Now())
							if err != nil {
								PrintCentre(err.Error(), "red")
							} else {
								chatNode.printChatRecords(infos)
							}
							fmt.Println(separator)
							PrintCentre("Press enter to refresh the chat records.", "magenta")
							PrintCentre("Press left arrow to return to superior page.", "magenta")
							PrintCentre("Type messages to chat.", "magenta")
							control := Scan('\n')
							if control == "" {
								continue
							} else if control == leftArrow {
								flush = true
								break
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
					return "HomePage"
				} else if consoleCursorIndex == 3 {
					return "Help"
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

func (chatNode *ChatNode) Help() string {
	fmt.Println(flushPadding)
	PrintCentre("To be developed", "red")
	fmt.Println(separator)
	PrintCentre("Type anything to return home page.", "magenta")
	Scan('\n')
	return "HomePage"
}

func (chatNode *ChatNode) Quit() {
	fmt.Println(flushPadding)
	PrintCentre("Goodbye!", "yellow")
}
