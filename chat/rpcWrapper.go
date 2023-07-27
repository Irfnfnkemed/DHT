package chat

type Null struct{}

type RPCWrapper struct {
	chatNode *ChatNode
}

func (wrapper *RPCWrapper) AcceptFriendRequest(friendName string, _ *Null) error {
	wrapper.chatNode.AcceptFriendRequest(friendName)
	return nil
}

func (wrapper *RPCWrapper) SendBackFriendRequest(pair SendBackPair, _ *Null) error {
	return wrapper.chatNode.SendBackFriendRequest(pair)
}

func (wrapper *RPCWrapper) AcceptInvitation(pair InvitationPair, _ *Null) error {
	return wrapper.chatNode.AcceptInvitation(pair)
}
