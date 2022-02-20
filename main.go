package letschat

import (
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type msg struct {
	ID, From, To, Txt string
	Time              int64
	Pri               bool
}

func (m *msg) toJSON() string {
	b, err := json.Marshal(m)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(b)
}

var msgID chan string

func genMsg(from, to, txt string, pri bool) *msg {
	return &msg{
		<-msgID,
		from,
		to,
		txt,
		time.Now().Unix(),
		pri,
	}
}

type client struct {
	ID, IP string
	ch     chan *msg
	kick   chan struct{}
	users  chan string
}

func (c *client) toJSON() string {
	b, err := json.Marshal(c)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(b)
}

var (
	msgLogs   *list.List
	msgLogsMu sync.RWMutex
)

var (
	clientsMu   sync.Mutex
	clients     []*client
	clientsChan chan []*client

	enter    chan *client
	leave    chan string // client id
	messages chan *msg

	logChan chan string
)

func init() {
	msgLogs = list.New()

	ts := strconv.FormatInt(time.Now().Unix(), 36)
	var start int64
	msgID = make(chan string)

	go func() {
		for {
			msgID <- fmt.Sprintf("%s-%d", ts, start)
			start++
		}
	}()

	logChan = make(chan string)

	go func() {
		for {
			s, ok := <-logChan
			if !ok {
				return
			}
			log.Println(s)
		}
	}()
}

func Start() {
	go clientsLoop()
	StartHTTP()
}

func clientsLoop() {
	messages = make(chan *msg)
	enter = make(chan *client)
	leave = make(chan string)

	for {
		select {
		case m := <-messages:
			recordLog(m)
			if m.Pri {
				for _, cli := range clients {
					if cli.ID == m.To || cli.ID == m.From {
						go sendTo(cli.ch, m)
					}
				}
				continue
			}

			for _, cli := range clients {
				go sendTo(cli.ch, m)
			}

		case cli := <-enter:
			log.Printf("enter: %s %s\n", cli.ID, cli.IP)
			if old := getClientByID(cli.ID); old != nil {
				close(old.kick)
			}
			clients = append(clients, cli)
			boradcaseUsers()

		case id := <-leave:
			log.Printf("leave: %s\n", id)

			removeClientByID(id)
			boradcaseUsers()
		}

	}
}

func removeClientByID(id string) {
	index := -1

	for i, cli := range clients {
		if cli.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	clients = append(clients[:index], clients[index+1:]...)
}

func getClientByID(id string) *client {
	for _, cli := range clients {
		if cli.ID == id {
			return cli
		}
	}
	return nil
}

func boradcaseUsers() {
	arr := make([]string, len(clients))

	for i, cli := range clients {
		arr[i] = cli.toJSON()
	}

	str := "[" + strings.Join(arr, ",") + "]"

	for _, c := range clients {
		go func(c *client) {
			c.users <- str
		}(c)
	}

}

func sendTo(ch chan<- *msg, m *msg) {
	select {
	case ch <- m:
	case <-time.After(10 * time.Second):
	}
}

func login(cli *client) {
	enter <- cli
}

func logout(id string) {
	leave <- id
}

const max = 100

func recordLog(m *msg) {
	log.Println(m.toJSON())

	msgLogsMu.Lock()
	msgLogs.PushBack(m)
	for msgLogs.Len() > 100 {
		msgLogs.Remove(msgLogs.Front())
	}
	msgLogsMu.Unlock()
}

func getLog(id string) []*msg {
	msgLogsMu.RLock()
	defer msgLogsMu.RUnlock()

	list := make([]*msg, 0, msgLogs.Len())

	for e := msgLogs.Front(); e != nil; e = e.Next() {
		m := e.Value.(*msg)
		if !m.Pri || m.From == id || m.To == id {
			list = append(list, m)
		}
	}

	return list
}
