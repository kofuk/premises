package msgrouter

import (
	"log"
	"sync"

	"golang.org/x/exp/slices"
)

type Message struct {
	Type     string `json:"type"`
	UserData string `json:"user_data"`
}

type MsgRouter struct {
	clients []chan Message
	m       sync.Mutex
}

func NewMsgRouter() *MsgRouter {
	return &MsgRouter{}
}

func (self *MsgRouter) DispatchMessage(msg Message) {
	self.m.Lock()
	defer self.m.Unlock()

	log.Println(msg)

	for _, client := range self.clients {
		client <- msg
	}
}

func (self *MsgRouter) Subscribe() chan Message {
	channel := make(chan Message, 8)
	self.m.Lock()
	defer self.m.Unlock()
	self.clients = append(self.clients, channel)
	return channel
}

func (self *MsgRouter) Unsubscribe(client chan Message) {
	self.m.Lock()
	defer self.m.Unlock()
	slices.DeleteFunc(self.clients, func(c chan Message) bool {
		return c == client
	})
}
