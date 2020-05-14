package main

import (
	"fmt"
	"time"

	"github.com/aurawing/auramq"
	"github.com/aurawing/auramq/msg"
	"github.com/aurawing/auramq/ws"
	client "github.com/aurawing/auramq/ws/cli"
)

func main() {
	router := auramq.NewRouter(1024)
	go router.Run()
	broker := ws.NewBroker(router, ":8080", true, auth, 0, 0, 0, 0, 0, 0)
	broker.Run()

	cli1, err := client.Connect("ws://127.0.0.1:8080/ws", callback, &msg.AuthReq{Id: "aaa", Credential: []byte("welcome")}, []string{"test"}, 0, 0, 0, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	go func() {
		cli1.Run()
	}()

	cli2, err := client.Connect("ws://127.0.0.1:8080/ws", callback, &msg.AuthReq{Id: "bbb", Credential: []byte("welcome")}, []string{"test"}, 0, 0, 0, 0)
	if err != nil {
		fmt.Println(err)
	}
	go func() {
		cli2.Run()
	}()

	b := cli1.Send("bbb", []byte("hahaha"))
	fmt.Println(b)
	time.Sleep(20 * time.Second)
	broker.Close()
	time.Sleep(100 * time.Second)
}

func callback(msg *msg.Message) {
	fmt.Printf("from %s to %s: %s\n", msg.Sender, msg.Destination, string(msg.Content))
}

func auth(b *msg.AuthReq) bool {
	if string(b.Credential) == "welcome" {
		return true
	} else {
		return false
	}
}