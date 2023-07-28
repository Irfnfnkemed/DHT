package chat

import (
	"bufio"
	"dht/chord"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const groupIdValidTime = 5 * time.Minute

type ChatNode struct {
	node                  *chord.Node
	name                  string
	friendList            map[string]string //名字->ip
	friendLock            sync.RWMutex
	groups                map[string]([]GroupChatRecord) //名字->群聊信息
	groupsLock            sync.RWMutex
	friendRequest         map[string]Null
	friendRequestLock     sync.RWMutex
	sentFriendRequest     map[string]string
	sentFriendRequestLock sync.RWMutex
	invitation            map[string]([]InvitationPair)
	invitationLock        sync.RWMutex
	start                 chan bool
	quit                  chan bool
}

type GroupChatRecord struct {
	GroupStartTime time.Time
	GroupSeed      string
}

type InfoRecord struct {
	FromName string    `json:"FromName"`
	SendTime time.Time `json:"SendTime"`
	Info     string    `json:"Info"`
}

type NamePair struct {
	FromName string
	ToName   string
}

type SendBackPair struct {
	FromName string
	Agree    bool
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
func (chatNode *ChatNode) Login(name, ip, knownIp string) error {
	chatNode.name = name
	chatNode.start = make(chan bool, 1)
	chatNode.quit = make(chan bool, 1)
	chatNode.friendList = make(map[string]string)
	chatNode.groups = make(map[string][]GroupChatRecord)
	chatNode.friendRequest = make(map[string]Null)
	chatNode.invitation = make(map[string][]InvitationPair)
	chatNode.sentFriendRequest = make(map[string]string)
	chatNode.node = new(chord.Node)
	ok := chatNode.node.Init(ip)
	if !ok {
		return errors.New("Init error.")
	}
	chatNode.node.Run()
	if knownIp == "" {
		chatNode.node.Create()
	} else {
		ok = chatNode.node.Join(knownIp)
		if !ok {
			return errors.New("Join error.")
		}
	}
	chatNode.node.RPC.Register("Chat", &RPCWrapper{chatNode})
	for true {
		ok, _ := chatNode.node.Get(name)
		if ok {
			fmt.Println("Name existed, please change a new one.\nType: ")
			reader := bufio.NewReader(os.Stdin)
			name, _ = reader.ReadString('\n')
			name = strings.TrimRight(name, "\n")
		} else {
			break
		}
	}
	chord.Setmode("overwrite")
	for i := 1; i <= 5; i++ {
		ok = chatNode.node.Put(name, ip)
		if ok {
			return nil
		}
	}
	return errors.New("IP put error.")
}

// 登出
func (chatNode *ChatNode) LogOut() {
	chatNode.node.Quit()
}

// 向用户发送好友请求，并将请求存入已发送记录中
func (chatNode *ChatNode) SendFriendRequest(friendName string) error {
	if friendName == chatNode.name {
		return errors.New("Cannot add myself as friend.")
	}
	chatNode.friendLock.RLock()
	friendIp, ok := chatNode.friendList[friendName]
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
	friendIp, ok = chatNode.sentFriendRequest[friendName]
	chatNode.sentFriendRequestLock.RUnlock()
	if ok && friendIp != "Accepted" && friendIp != "Rejected" {
		return errors.New("You have sent the request to this user. Please wait friend to confirm.")
	}
	ok, friendIp = chatNode.node.Get(friendName)
	if !ok {
		return errors.New("Not existed user.")
	}
	err := chatNode.node.RPC.RemoteCall(friendIp, "Chat.AcceptFriendRequest", chatNode.name, &Null{})
	if err != nil {
		return err
	}
	chatNode.sentFriendRequestLock.Lock()
	chatNode.sentFriendRequest[friendName] = friendIp
	chatNode.sentFriendRequestLock.Unlock()
	return nil
}

// 接收好友请求，存入待确认列表
func (chatNode *ChatNode) AcceptFriendRequest(friendName string) {
	chatNode.friendRequestLock.Lock()
	chatNode.friendRequest[friendName] = Null{}
	chatNode.friendRequestLock.Unlock()
}

// 确认待确认列表中的好友请求
func (chatNode *ChatNode) CheckFriendRequest(friendName string, agree bool) error {
	chatNode.friendRequestLock.Lock()
	delete(chatNode.friendRequest, friendName)
	chatNode.friendRequestLock.Unlock()
	ok, friendIp := chatNode.node.Get(friendName)
	if !ok {
		return errors.New("Get IP error.")
	}
	err := chatNode.node.RPC.RemoteCall(friendIp, "Chat.SendBackFriendRequest", SendBackPair{chatNode.name, agree}, &Null{})
	if err != nil {
		return err
	}
	if agree {
		chatNode.friendLock.Lock()
		chatNode.friendList[friendName] = friendIp
		chatNode.friendLock.Unlock()
	}
	return nil
}

// 向好友请求发起者返回自己对请求的确认结果
func (chatNode *ChatNode) SendBackFriendRequest(pair SendBackPair) error {
	friendIp, ok := chatNode.sentFriendRequest[pair.FromName]
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
		chatNode.friendList[pair.FromName] = friendIp
		chatNode.friendLock.Unlock()
		return nil
	}
	return errors.New("Request was rejeceted.")
}

// 得到好友列表
func (chatNode *ChatNode) GetFriendList() []string {
	friendList := []string{}
	chatNode.friendLock.RLock()
	defer chatNode.friendLock.RUnlock()
	for name := range chatNode.friendList {
		friendList = append(friendList, name)
	}
	return friendList
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
	ip, ok := chatNode.friendList[friendName]
	chatNode.friendLock.RUnlock()
	if !ok {
		return errors.New("Not existed friend.")
	}
	err := chatNode.node.RPC.RemoteCall(ip, "Chat.AcceptInvitation",
		InvitationPair{chatNode.name, friendName, groupChatName, groupChat.GroupSeed, groupChat.GroupStartTime}, &Null{})
	if err != nil {
		return err
	}
	return nil
}

// 接受群聊邀请，加入待确认列表中
func (chatNode *ChatNode) AcceptInvitation(pair InvitationPair) error {
	chatNode.groupsLock.RLock()
	if chatNode.groups[pair.GroupChatName] != nil {
		for _, group := range chatNode.groups[pair.GroupChatName] {
			if group.GroupSeed == pair.GroupSeed {
				return errors.New("User has been in that group chat already.")
			}
		}
	}
	chatNode.groupsLock.RUnlock()
	chatNode.invitationLock.RLock()
	if chatNode.invitation[pair.GroupChatName] != nil {
		for _, group := range chatNode.invitation[pair.GroupChatName] {
			if group.GroupSeed == pair.GroupSeed {
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
	jsonInfo, err := json.Marshal(InfoRecord{chatNode.name, sendTime, info})
	if err != nil {
		return err
	}
	chord.Setmode("append")
	ok := chatNode.node.Put(getGroupIp(groupChat.GroupSeed, groupChat.GroupStartTime, sendTime), string(jsonInfo)+"\n")
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
		_, jsonInfo := chatNode.node.Get(groupIpNow)
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
		ok, _ := chatNode.node.Get(getGroupIp(groupChat.GroupSeed, groupChat.GroupStartTime, timeNow))
		if ok {
			return timeNow, nil
		}
	}
	return time.Time{}, errors.New("No more records.")
}
