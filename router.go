package auramq

import (
	"errors"
	"log"
	"sync"

	"github.com/aurawing/auramq/msg"
	"github.com/fatih/set"
)

//Router routing message to subscriber
type Router struct {
	subscribers map[string]Subscriber
	rtable      map[string]set.Interface
	rrtable     map[Subscriber]set.Interface
	broadcast   chan *msg.Message
	done        chan struct{}
	lock        sync.RWMutex
}

//NewRouter create a new router instance
func NewRouter(bufferSize int) *Router {
	return &Router{
		subscribers: make(map[string]Subscriber),
		rtable:      make(map[string]set.Interface),
		rrtable:     make(map[Subscriber]set.Interface),
		broadcast:   make(chan *msg.Message, bufferSize),
		done:        make(chan struct{}),
	}
}

//Register register topics for subscriber
func (router *Router) Register(client Subscriber, topics []string) error {
	router.lock.Lock()
	defer router.lock.Unlock()
	if _, ok := router.subscribers[client.ID()]; ok {
		return errors.New("client ID conflict")
	}
	router.subscribers[client.ID()] = client
	if router.rrtable[client] == nil {
		router.rrtable[client] = set.New(set.NonThreadSafe)
	}
	s := set.New(set.NonThreadSafe)
	for _, t := range topics {
		s.Add(t)
	}
	intersect := set.Difference(s, router.rrtable[client])
	for _, t := range intersect.List() {
		router.rrtable[client].Add(t)
	}
	for _, topic := range intersect.List() {
		if router.rtable[topic.(string)] == nil {
			router.rtable[topic.(string)] = set.New(set.NonThreadSafe)
		}
		router.rtable[topic.(string)].Add(client)
	}
	return nil
}

//UnregisterSubscriber unregister all topics for subscriber
func (router *Router) UnregisterSubscriber(client Subscriber) {
	router.lock.Lock()
	defer router.lock.Unlock()
	if _, ok := router.rrtable[client]; !ok {
		return
	}
	topics := router.rrtable[client].List()
	topicList := make([]string, 0)
	for _, t := range topics {
		topicList = append(topicList, t.(string))
	}
	router.unregister(client, topicList)
}

//Unregister unregister topics for subscriber
func (router *Router) Unregister(client Subscriber, topics []string) {
	router.lock.Lock()
	defer router.lock.Unlock()
	router.unregister(client, topics)
}

func (router *Router) unregister(client Subscriber, topics []string) {
	if _, ok := router.rrtable[client]; !ok {
		return
	}
	if router.rrtable[client].Size() == 0 {
		delete(router.rrtable, client)
		return
	}
	for _, topic := range topics {
		router.rtable[topic].Remove(client)
		if router.rtable[topic].Size() == 0 {
			delete(router.rtable, topic)
		}
		router.rrtable[client].Remove(topic)
	}
	if router.rrtable[client].Size() == 0 {
		delete(router.rrtable, client)
		delete(router.subscribers, client.ID())
	}
}

//Publish publish message to a topic
func (router *Router) Publish(msg *msg.Message) {
	router.broadcast <- msg
}

//Run start router
func (router *Router) Run() {
OUT:
	for {
		select {
		case msg := <-router.broadcast:
			if msg.Type == BROADCAST && router.rtable[msg.Destination] != nil {
				for _, client := range router.rtable[msg.Destination].List() {
					cli := client.(Subscriber)
					cli.Send(msg)
					// if !cli.Send(msg) {
					// 	cli.Close()
					// 	router.UnregisterSubscriber(cli)
					// }
				}
			} else if msg.Type == P2P {
				log.Printf("received P2P message from %s to %s\n", msg.Sender, msg.Destination)
				if client, ok := router.subscribers[msg.Destination]; ok {
					cli := client.(Subscriber)
					cli.Send(msg)
				}
			}
		case _ = <-router.done:
			break OUT
		}
	}
	close(router.broadcast)
	close(router.done)
	for cli := range router.rrtable {
		cli.Close()
	}
}

//Close shutdown router
func (router *Router) Close() {
	router.done <- struct{}{}
}
