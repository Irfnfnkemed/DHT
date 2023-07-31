package chat

import (
	"dht/chord"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

const groupIdValidTime = 5 * time.Minute
const trySendCircle = 5 * time.Second

type ChatNode struct {
	node                  *chord.Node
	name                  string
	password              string
	accountSeed           string
	friendList            map[string]GroupChatRecord // 名字->私聊信息
	friendLock            sync.RWMutex
	groups                map[string]([]GroupChatRecord) //名字->群聊信息
	groupsLock            sync.RWMutex
	friendRequest         map[string]SendBackPair
	friendRequestLock     sync.RWMutex
	sentFriendRequest     map[string]string
	sentFriendRequestLock sync.RWMutex
	invitation            map[string]([]InvitationPair)
	invitationLock        sync.RWMutex
	sentInvitation        map[string]([]InvitationPair) // 发出但对方还未接收的群聊邀请
	sentInvitationLock    sync.RWMutex
	online                bool
}

type AccountRecord struct {
	Online      bool
	IP          string
	AccountSeed string
}

type ChatNodeRecord struct {
	FriendList        map[string]GroupChatRecord
	Groups            map[string]([]GroupChatRecord)
	FriendRequest     map[string]SendBackPair
	SentFriendRequest map[string]string
	Invitation        map[string]([]InvitationPair)
	SentInvitation    map[string]([]InvitationPair)
}

type GroupChatRecord struct {
	GroupStartTime time.Time
	GroupSeed      string
}

type InfoRecord struct {
	FromName string
	SendTime time.Time
	Info     string
}

type NamePair struct {
	FromName string
	ToName   string
}

type SendBackPair struct {
	Agree         bool
	FromName      string
	ChatSeed      string
	ChatStartTime time.Time
}

type InvitationPair struct {
	FromName       string
	ToName         string
	GroupChatName  string
	GroupSeed      string
	GroupStartTime time.Time
}

// 初始化
func init() {
	err := setConsoleWidth()
	if err != nil {
		fmt.Println("Get console width error.")
	}
	setStrings()
}

// 登录
func (chatNode *ChatNode) Login(name, ip, password, knownIp string, register bool) error {
	chatNode.node = new(chord.Node)
	ok := chatNode.node.Init(ip)
	if !ok {
		return errors.New("Init error.")
	}
	chatNode.node.Run()
	chatNode.node.RPC.Register("Chat", &RPCWrapper{chatNode})
	if register {
		chatNode.name = name
		chatNode.password = password
		chatNode.accountSeed = randString(60)
		chatNode.friendList = make(map[string]GroupChatRecord)
		chatNode.groups = make(map[string][]GroupChatRecord)
		chatNode.friendRequest = make(map[string]SendBackPair)
		chatNode.invitation = make(map[string][]InvitationPair)
		chatNode.sentFriendRequest = make(map[string]string)
		chatNode.sentInvitation = make(map[string][]InvitationPair)
	}
	if knownIp == "" {
		done := make(chan bool, 1)
		go func() {
			chatNode.node.Create()
			done <- true
		}()
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			return errors.New("Create error.")
		}
	} else {
		ok = chatNode.node.Join(knownIp)
		if !ok {
			return errors.New("Join error.")
		}
	}
	time.Sleep(500 * time.Millisecond) //使得数据转移
	if register {
		for true {
			ok, _ := chatNode.node.Get(name)
			if ok {
				PrintCentre("Name existed, please change a new one.", "red")
				PrintCentre("Type:", "yellow")
				name = Scan('\n')
				chatNode.name = name
			} else {
				break
			}
		}
	} else {
		for {
			ok, accountString := chatNode.node.Get(name)
			if ok {
				accountRecord := AccountRecord{}
				err := json.Unmarshal([]byte(accountString), &accountRecord)
				if err != nil {
					chatNode.node.Quit()
					fmt.Println(err)
					fmt.Println(accountRecord)
					return errors.New("Account get error.")
				}
				if accountRecord.Online {
					chatNode.node.Quit()
					return errors.New("The account is online on other device now!")
				}
				chatNode.name = name
				chatNode.accountSeed = accountRecord.AccountSeed
				break
			} else {
				PrintCentre("The account isn't exsited, please check again!", "red")
				PrintCentre("Type account name:", "yellow")
				name = Scan('\n')
			}
		}
		for {
			ok, chatNodeRecordString := chatNode.node.Get(chatNode.accountSeed + password)
			if ok {
				chatNodeRecord := ChatNodeRecord{}
				err := json.Unmarshal([]byte(chatNodeRecordString), &chatNodeRecord)
				if err != nil {
					chatNode.node.Quit()
					fmt.Println(err)
					fmt.Println(chatNodeRecordString)
					return errors.New("Account info get error.")
				}
				chatNode.password = password
				chatNode.friendList = chatNodeRecord.FriendList
				chatNode.groups = chatNodeRecord.Groups
				chatNode.friendRequest = chatNodeRecord.FriendRequest
				chatNode.invitation = chatNodeRecord.Invitation
				chatNode.sentFriendRequest = chatNodeRecord.SentFriendRequest
				chatNode.sentInvitation = chatNodeRecord.SentInvitation
				break
			} else {
				PrintCentre("The password is wrong! please check again!", "red")
				PrintCentre("Type password:", "yellow")
				password = Scan('\n')
			}
		}
	}
	jsonInfo, _ := json.Marshal(AccountRecord{true, ip, chatNode.accountSeed})
	ok = chatNode.node.Put(name, string(jsonInfo))
	if !ok {
		chatNode.node.Quit()
		return errors.New("Account put error.")
	}
	chatNode.online = true
	chatNode.RestartSendTry()
	return nil
}

// 登出
func (chatNode *ChatNode) LogOut() {
	chatNode.online = false
	jsonInfo, _ := json.Marshal(AccountRecord{false, chatNode.node.IP, chatNode.accountSeed})
	ok := chatNode.node.Put(chatNode.name, string(jsonInfo))
	if !ok {
		PrintCentre("Save account error!", "red")
	}
	jsonInfo, _ = json.Marshal(ChatNodeRecord{chatNode.friendList, chatNode.groups,
		chatNode.friendRequest, chatNode.sentFriendRequest, chatNode.invitation, chatNode.sentInvitation})
	ok = chatNode.node.Put(chatNode.accountSeed+chatNode.password, string(jsonInfo))
	if !ok {
		PrintCentre("Save account error!", "red")
	}
	time.Sleep(1 * time.Second)
	chatNode.node.Quit()
}

// 向用户发送好友请求，并将请求存入已发送记录中
func (chatNode *ChatNode) SendFriendRequest(friendName string) error {
	if friendName == chatNode.name {
		return errors.New("Cannot add myself as friend.")
	}
	chatNode.friendLock.RLock()
	_, ok := chatNode.friendList[friendName]
	chatNode.friendLock.RUnlock()
	if ok {
		return errors.New("The user has been your friend.")
	}
	chatNode.friendRequestLock.RLock()
	_, ok = chatNode.friendRequest[friendName]
	chatNode.friendRequestLock.RUnlock()
	if ok {
		return errors.New("You have received the request from this user. Please confirm the friend request.")
	}
	chatNode.sentFriendRequestLock.RLock()
	status, ok := chatNode.sentFriendRequest[friendName]
	chatNode.sentFriendRequestLock.RUnlock()
	if ok && status != "Accepted" && status != "Rejected" {
		return errors.New("You have sent the request to this user. Please wait friend to confirm.")
	}
	go chatNode.TrySendFriendRequest(friendName)
	time.Sleep(250 * time.Millisecond)
	return nil
}

// 接收好友请求，存入待确认列表
func (chatNode *ChatNode) AcceptFriendRequest(friendName string) {
	chatNode.friendRequestLock.Lock()
	chatNode.friendRequest[friendName] = SendBackPair{} //结构体空值，表示尚未确认
	chatNode.friendRequestLock.Unlock()
}

// 确认待确认列表中的好友请求
func (chatNode *ChatNode) CheckFriendRequest(friendName string, agree bool) error {
	chatNode.friendRequestLock.Lock()
	delete(chatNode.friendRequest, friendName)
	chatNode.friendRequestLock.Unlock()
	privateChat := GroupChatRecord{time.Now(), randString(60)}
	go chatNode.TrySendBackFriendRequest(friendName, SendBackPair{agree, chatNode.name, privateChat.GroupSeed, privateChat.GroupStartTime})
	time.Sleep(250 * time.Millisecond)
	if agree {
		chatNode.friendLock.Lock()
		chatNode.friendList[friendName] = privateChat
		chatNode.friendLock.Unlock()
	}
	return nil
}

// 向好友请求发起者返回自己对请求的确认结果
func (chatNode *ChatNode) SendBackFriendRequest(pair SendBackPair) error {
	chatNode.sentFriendRequestLock.RLock()
	_, ok := chatNode.sentFriendRequest[pair.FromName]
	chatNode.sentFriendRequestLock.RUnlock()
	if !ok {
		return errors.New("Sent friend request error.")
	}
	chatNode.sentFriendRequestLock.Lock()
	if pair.Agree {
		chatNode.sentFriendRequest[pair.FromName] = "Accepted"
	} else {
		chatNode.sentFriendRequest[pair.FromName] = "Rejected"
	}
	chatNode.sentFriendRequestLock.Unlock()
	if pair.Agree {
		chatNode.friendLock.Lock()
		chatNode.friendList[pair.FromName] = GroupChatRecord{pair.ChatStartTime, pair.ChatSeed}
		chatNode.friendLock.Unlock()
		return nil
	}
	return errors.New("Request was rejeceted.")
}

// 得到好友列表和私聊列表
func (chatNode *ChatNode) GetFriendList() ([]string, []GroupChatRecord) {
	friendList := []string{}
	privateChatList := []GroupChatRecord{}
	chatNode.friendLock.RLock()
	defer chatNode.friendLock.RUnlock()
	for name, privateChat := range chatNode.friendList {
		friendList = append(friendList, name)
		privateChatList = append(privateChatList, privateChat)
	}
	return friendList, privateChatList
}

// 删除已发送的好友请求记录
func (chatNode *ChatNode) DeleteSentRequest(friendName string) error {
	chatNode.sentFriendRequestLock.Lock()
	defer chatNode.sentFriendRequestLock.Unlock()
	if chatNode.sentFriendRequest[friendName] == "Accepted" || chatNode.sentFriendRequest[friendName] == "Rejected" {
		delete(chatNode.sentFriendRequest, friendName)
		return nil
	}
	return errors.New("Can't be deleted.")
}

// 创建群聊
func (chatNode *ChatNode) CreateChatGroup(groupChatName string) {
	chatNode.groupsLock.Lock()
	if chatNode.groups[groupChatName] == nil {
		chatNode.groups[groupChatName] = make([]GroupChatRecord, 0)
	}
	chatNode.groups[groupChatName] = append(chatNode.groups[groupChatName],
		GroupChatRecord{time.Now(), randString(60)})
	chatNode.groupsLock.Unlock()
}

// 邀请好友加入群聊
func (chatNode *ChatNode) InviteFriend(friendName, groupChatName string, groupChat GroupChatRecord) error {
	chatNode.friendLock.RLock()
	_, ok := chatNode.friendList[friendName]
	chatNode.friendLock.RUnlock()
	if !ok {
		return errors.New("Not existed friend.")
	}
	chatNode.sentInvitationLock.RLock()
	sentInvitations, ok := chatNode.sentInvitation[groupChatName]
	chatNode.sentInvitationLock.RUnlock()
	if ok {
		for _, sentInvitation := range sentInvitations {
			if sentInvitation.GroupSeed == groupChat.GroupSeed && sentInvitation.ToName == friendName {
				return errors.New("You have sent invitation to this friend. Please wait him/her to confirm.")
			}
		}
	}
	pair := InvitationPair{chatNode.name, friendName, groupChatName, groupChat.GroupSeed, groupChat.GroupStartTime}
	chatNode.sentInvitationLock.Lock()
	_, ok = chatNode.sentInvitation[groupChatName]
	if !ok {
		chatNode.sentInvitation[groupChatName] = make([]InvitationPair, 0)
	}
	chatNode.sentInvitation[groupChatName] = append(chatNode.sentInvitation[groupChatName], pair)
	chatNode.sentInvitationLock.Unlock()
	err := chatNode.TryInvite(pair)
	if err == nil {
		return nil
	}
	if err.Error() == "User has been in that group chat already." ||
		err.Error() == "User has received the invitation. Please wait him/her to confirm." {
		return err
	}
	go func() {
		for chatNode.online {
			err := chatNode.TryInvite(pair)
			if err == nil || err.Error() == "User has been in that group chat already." ||
				err.Error() == "User has received the invitation. Please wait him/her to confirm." {
				return
			}
			time.Sleep(trySendCircle)
		}
	}()
	return errors.New("Pending.")
}

// 接受群聊邀请，加入待确认列表中
func (chatNode *ChatNode) AcceptInvitation(pair InvitationPair) error {
	chatNode.groupsLock.RLock()
	if chatNode.groups[pair.GroupChatName] != nil {
		for _, group := range chatNode.groups[pair.GroupChatName] {
			if group.GroupSeed == pair.GroupSeed {
				chatNode.groupsLock.RUnlock()
				return errors.New("User has been in that group chat already.")
			}
		}
	}
	chatNode.groupsLock.RUnlock()
	chatNode.invitationLock.RLock()
	if chatNode.invitation[pair.GroupChatName] != nil {
		for _, group := range chatNode.invitation[pair.GroupChatName] {
			if group.GroupSeed == pair.GroupSeed {
				chatNode.invitationLock.RUnlock()
				return errors.New("User has received the invitation. Please wait him/her to confirm.")
			}
		}
	}
	chatNode.invitationLock.RUnlock()
	chatNode.invitationLock.Lock()
	if chatNode.invitation[pair.GroupChatName] == nil {
		chatNode.invitation[pair.GroupChatName] = make([]InvitationPair, 0)
	}
	chatNode.invitation[pair.GroupChatName] =
		append(chatNode.invitation[pair.GroupChatName], pair)
	chatNode.invitationLock.Unlock()
	return nil
}

// 确认待确认列表中的群聊邀请
func (chatNode *ChatNode) CheckInvitation(pair InvitationPair, agree bool) error {
	chatNode.invitationLock.Lock()
	for i, group := range chatNode.invitation[pair.GroupChatName] {
		if group.GroupSeed == pair.GroupSeed {
			chatNode.invitation[pair.GroupChatName] =
				append(chatNode.invitation[pair.GroupChatName][:i], chatNode.invitation[pair.GroupChatName][i+1:]...) // 删除记录
			if len(chatNode.invitation[pair.GroupChatName]) == 0 {
				delete(chatNode.invitation, pair.GroupChatName)
			}
		}
	}
	chatNode.invitationLock.Unlock()
	if agree {
		chatNode.groupsLock.Lock()
		if chatNode.groups[pair.GroupChatName] == nil {
			chatNode.groups[pair.GroupChatName] = make([]GroupChatRecord, 0)
		}
		chatNode.groups[pair.GroupChatName] = append(chatNode.groups[pair.GroupChatName],
			GroupChatRecord{pair.GroupStartTime, pair.GroupSeed})
		chatNode.groupsLock.Unlock()
		time.Sleep(100 * time.Millisecond)
		err := chatNode.SendChatInfo("Hello everyone! I was invited by "+pair.FromName,
			GroupChatRecord{pair.GroupStartTime, pair.GroupSeed})
		return err
	}
	return nil
}

// 向群聊发送聊天信息
func (chatNode *ChatNode) SendChatInfo(info string, groupChat GroupChatRecord) error {
	sendTime := time.Now()
	jsonInfo, _ := json.Marshal(InfoRecord{chatNode.name, sendTime, info})
	ok := chatNode.node.PutMode(getGroupIp(groupChat.GroupSeed, groupChat.GroupStartTime, sendTime), string(jsonInfo)+"\n", "append")
	if !ok {
		return errors.New("Send info error.")
	}
	return nil
}

// 得到目标时间区间内的群聊消息
func (chatNode *ChatNode) GetChatInfo(groupChat GroupChatRecord, beginTime, endTime time.Time) ([]InfoRecord, error) {
	groupIpEnd := getGroupIp(groupChat.GroupSeed, groupChat.GroupStartTime, endTime)
	groupIpNow := ""
	infos := []InfoRecord{}
	for nowTime := beginTime; groupIpEnd != groupIpNow; nowTime = nowTime.Add(groupIdValidTime) {
		groupIpNow = getGroupIp(groupChat.GroupSeed, groupChat.GroupStartTime, nowTime)
		_, jsonInfo := chatNode.node.GetMode(groupIpNow, "append")
		infoTmp, err := parseToInfoRecord(jsonInfo)
		if err != nil {
			return infos, err
		}
		infos = append(infos, infoTmp...)
	}
	return infos, nil
}

// 得到群聊更早的有聊天记录的时间(群聊信息是按时间一段段存储的)
func (chatNode *ChatNode) GetEarlierChatInfoTime(groupChat GroupChatRecord, beginTime time.Time) (time.Time, error) {
	timeNow := beginTime.Add(-groupIdValidTime)
	for timeNow.After(groupChat.GroupStartTime) {
		ok, _ := chatNode.node.GetMode(getGroupIp(groupChat.GroupSeed, groupChat.GroupStartTime, timeNow), "append")
		if ok {
			return timeNow, nil
		}
	}
	return time.Time{}, errors.New("No more records.")
}

// 不断尝试发送好友请求，直至接收
func (chatNode *ChatNode) TrySendFriendRequest(friendName string) {
	chatNode.sentFriendRequestLock.Lock()
	chatNode.sentFriendRequest[friendName] = "Pending"
	chatNode.sentFriendRequestLock.Unlock()
	for chatNode.online {
		friendIp, _, err := chatNode.getUserAccount(friendName)
		if err != nil {
			time.Sleep(trySendCircle)
			continue
		}
		err = chatNode.node.RPC.RemoteCall(friendIp, "Chat.AcceptFriendRequest", chatNode.name, &Null{})
		if err == nil {
			chatNode.sentFriendRequestLock.Lock()
			chatNode.sentFriendRequest[friendName] = "To be confirmed"
			chatNode.sentFriendRequestLock.Unlock()
			return
		}
		time.Sleep(trySendCircle)
	}
}

// 不断尝试发送好友请求确认情况，直至接收
func (chatNode *ChatNode) TrySendBackFriendRequest(friendName string, pair SendBackPair) {
	chatNode.friendRequestLock.Lock()
	chatNode.friendRequest[friendName] = pair //表示已经确认，但对方还未接收
	chatNode.friendRequestLock.Unlock()
	for chatNode.online {
		friendIp, _, err := chatNode.getUserAccount(friendName)
		if err != nil {
			time.Sleep(trySendCircle)
			continue
		}
		err = chatNode.node.RPC.RemoteCall(friendIp, "Chat.SendBackFriendRequest", pair, &Null{})
		if err == nil {
			chatNode.friendRequestLock.Lock()
			delete(chatNode.friendRequest, friendName)
			chatNode.friendRequestLock.Unlock()
			return
		}
		time.Sleep(trySendCircle)
	}
}

func (chatNode *ChatNode) RestartSendTry() {
	sendList := []string{}
	chatNode.sentFriendRequestLock.RLock()
	for name, status := range chatNode.sentFriendRequest {
		if status == "Pending" {
			sendList = append(sendList, name)
		}
	}
	chatNode.sentFriendRequestLock.RUnlock()
	for _, name := range sendList {
		go chatNode.TrySendFriendRequest(name)
	}
	sendBackList := make(map[string]SendBackPair)
	chatNode.friendRequestLock.RLock()
	for name, sendBack := range chatNode.friendRequest {
		if sendBack.ChatSeed != "" {
			sendBackList[name] = sendBack
		}
	}
	chatNode.friendRequestLock.RUnlock()
	for name, sendBack := range sendBackList {
		go chatNode.TrySendBackFriendRequest(name, sendBack)
	}
	invitationList := []InvitationPair{}
	chatNode.sentInvitationLock.RLock()
	for _, invitations := range chatNode.sentInvitation {
		invitationList = append(invitationList, invitations...)
	}
	chatNode.sentInvitationLock.RUnlock()
	for _, pair := range invitationList {
		go func(pair InvitationPair) {
			for chatNode.online {
				err := chatNode.TryInvite(pair)
				if err == nil || err.Error() == "User has been in that group chat already." ||
					err.Error() == "User has received the invitation. Please wait him/her to confirm." {
					return
				}
				time.Sleep(trySendCircle)
			}
		}(pair)
	}

}

func (chatNode *ChatNode) TryInvite(pair InvitationPair) error {
	friendIp, _, err := chatNode.getUserAccount(pair.ToName)
	if err != nil {
		return err
	}
	err = chatNode.node.RPC.RemoteCall(friendIp, "Chat.AcceptInvitation", pair, &Null{})
	if err == nil || err.Error() == "User has been in that group chat already." ||
		err.Error() == "User has received the invitation. Please wait him/her to confirm." {
		chatNode.sentInvitationLock.Lock()
		sentInvitations, ok := chatNode.sentInvitation[pair.GroupChatName]
		if ok {
			for i, sentInvitation := range sentInvitations {
				if sentInvitation.GroupSeed == pair.GroupSeed && sentInvitation.ToName == pair.ToName {
					sentInvitations = append(sentInvitations[:i], sentInvitations[i+1:]...)
					break
				}
			}
		}
		if len(sentInvitations) == 0 {
			delete(chatNode.sentInvitation, pair.GroupChatName)
		} else {
			chatNode.sentInvitation[pair.GroupChatName] = sentInvitations
		}
		chatNode.sentInvitationLock.Unlock()
	}
	return err
}
