package msgrouter

import (
	"sync"

	"golang.org/x/exp/slices"
)

type Message struct {
	Type     string `json:"type"`
	Dispatch bool   `json:"dispatch"`
	UserData string `json:"user_data"`
}

type MsgRouter struct {
	clients    []*Client
	m          sync.Mutex
	latestMsgs map[string]*Message
}

type Client struct {
	C chan Message
}

func NewMsgRouter() *MsgRouter {
	return &MsgRouter{
		latestMsgs: make(map[string]*Message),
	}
}

func (self *MsgRouter) DispatchMessage(msg Message) {
	self.m.Lock()
	defer self.m.Unlock()

	self.latestMsgs[msg.Type] = &msg

	for _, client := range self.clients {
		client.C <- msg
	}
}

type SubscriptionOption func(c *Client, msgRouter *MsgRouter)

func NotifyLatest(msgType string) SubscriptionOption {
	return func(c *Client, msgRouter *MsgRouter) {
		if msgRouter.latestMsgs[msgType] != nil {
			c.C <- *msgRouter.latestMsgs[msgType]
		}
	}
}

func (self *MsgRouter) Subscribe(opts ...SubscriptionOption) *Client {
	client := &Client{
		C: make(chan Message, 8),
	}
	self.m.Lock()
	defer self.m.Unlock()

	for _, opt := range opts {
		opt(client, self)
	}

	self.clients = append(self.clients, client)

	return client
}

func (self *MsgRouter) Unsubscribe(client *Client) {
	self.m.Lock()
	defer self.m.Unlock()
	slices.DeleteFunc(self.clients, func(c *Client) bool {
		return c == client
	})
}
