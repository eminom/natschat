package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/nats-io/go-nats"
	"io"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"syscall"
)

const (
	TargetServer = "nats://192.168.18.1:4222"
	Subject      = "gchat"
	XHint        = `:\> `
)

type XMsg struct {
	Sender string `json:"sender"`
	Msg    string `json:"msg"`
}

type XChatter struct {
	myself  string
	pm      sync.Mutex
	enInput bool
}

func CreateChatter() *XChatter {
	return &XChatter{
		myself:  whoIsMe(),
		enInput: false,
	}
}

//~ the first one is
func (self *XChatter) doInLock(isOn bool, p ...interface{}) {
	self.pm.Lock()
	self.enInput = isOn
	if len(p) > 0 {
		if reflect.TypeOf(p[0]).Kind() == reflect.Func {
			p[0].(func())()
		}
	}
	self.pm.Unlock()
}

func (self *XChatter) emitMsg(msg string) []byte {
	b, err := json.Marshal(&XMsg{
		Sender: self.myself,
		Msg:    msg,
	})
	if err != nil {
		fmt.Println("Encoding error")
		panic(err)
	}
	var m2 XMsg
	err2 := json.Unmarshal(b, &m2)
	if err2 != nil {
		panic(err2)
	}
	return b
}

func (self *XChatter) getMsgHandler() func(*nats.Msg) {
	return func(m *nats.Msg) {
		self.pm.Lock()
		var msg XMsg
		err := json.Unmarshal(m.Data, &msg)
		if nil != err {
			fmt.Printf("%v\n", err)
		} else if msg.Sender != self.myself {
			if self.enInput {
				fmt.Printf("\n")
			}
			fmt.Printf("From %v:%v\n", msg.Sender, msg.Msg)
			if self.enInput {
				fmt.Printf(XHint)
			}
		}
		self.pm.Unlock()
	}
}

///~ Using a (most likely to be) unique name in cyberspace
func whoIsMe() string {
	ifs, err := net.Interfaces()
	if nil != err {
		fmt.Println("cannot get local ip")
		panic(err)
	}
	for _, inter := range ifs {
		mac := inter.HardwareAddr
		return fmt.Sprintf("%v[%d]", mac, rand.Intn(10))
	}
	panic("no mac detected")
}

func ReadLine(reader io.Reader, inputLineChan chan string) {
	buf := bufio.NewReader(reader)
	line, err := buf.ReadBytes('\n')
	for err == nil {
		line = bytes.TrimRight(line, "\n")
		if len(line) > 0 {
			if line[len(line)-1] == 13 { //'\r'
				line = bytes.TrimRight(line, "\r")
			}
			//return string(line) //~ Do not do output
			inputLineChan <- string(line)
		}
		line, err = buf.ReadBytes('\n')
	}
	if len(line) > 0 {
		inputLineChan <- string(line)
	}
}

func test1() {
	m1 := XMsg{"hola", "mundo"}
	b, _ := json.Marshal(&m1)
	var m2 XMsg
	fmt.Printf("Test b is:%v\n", string(b))
	err := json.Unmarshal(b, &m2)
	if err != nil {
		fmt.Printf("decode error\n")
		fmt.Println(err)
	} else {
		fmt.Printf("decode done\n")
		fmt.Println(m2)
	}
}

func main() {
	test1() // The test is passed.
	nc, err := nats.Connect(TargetServer)
	defer nc.Close()
	if err != nil {
		fmt.Printf("Error:%v\n", err)
		fmt.Printf("Quit")
		os.Exit(1)
	}

	talker := CreateChatter()
	nc.Subscribe(Subject, talker.getMsgHandler())

	osc := make(chan os.Signal, 1)
	signal.Notify(osc, os.Interrupt, syscall.Interrupt, os.SIGTERM)

	lineSwitcher := make(chan string)
	go ReadLine(os.Stdin, lineSwitcher)
A100:
	for {
		talker.doInLock(true, func() { fmt.Printf(XHint) })
		select {
		case newLine := <-lineSwitcher:
			//fmt.Scanf("%s\n", &inputText)
			inputText := strings.TrimRight(newLine, "\t \n")
			nc.Publish(Subject, talker.emitMsg(inputText))
		case <-osc:
			break A100
		}
		talker.doInLock(false)
	}
}
