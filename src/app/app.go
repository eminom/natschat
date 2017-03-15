package main

import (
	"fmt"
	"github.com/nats-io/go-nats"
	"os"
	"strings"
	"sync"
)

const (
	TargetServer = "nats://192.168.18.1:4222"
	Subject      = "gchat"
	XHint        = `:\> `
)

func main() {
	nc, err := nats.Connect(TargetServer)
	if err != nil {
		fmt.Printf("Error:%v\n", err)
		fmt.Printf("Quit")
		os.Exit(1)
	}
	//pCond := sync.NewCond()
	var pm sync.Mutex
	enInput := true
	nc.Subscribe(Subject, func(m *nats.Msg) {
		pm.Lock()
		if enInput {
			fmt.Printf("\n")
		}
		fmt.Printf("%s\n", string(m.Data))
		if enInput {
			fmt.Printf(XHint)
		}
		pm.Unlock()
	})
	for {
		pm.Lock()
		enInput = true
		pm.Unlock()
		fmt.Printf(XHint)
		var s string
		fmt.Scanf("%s\n", &s)
		pm.Lock()
		enInput = false
		pm.Unlock()
		s = strings.TrimRight(s, " \t\n")
		nc.Publish(Subject, []byte(s))
	}
}
