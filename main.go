package chat

import (
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
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
	id    string
	ch    chan *msg
	kick  chan struct{}
	users chan []string
}

var (
	msgLogs   *list.List
	msgLogsMu sync.RWMutex
)

var (
	clientsMu   sync.Mutex
	clients     map[string]*client
	clientsChan chan map[string]*client

	enter    chan *client
	leave    chan string // client id
	messages chan *msg
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
}

func Start() {
	go clientsLoop()
	StartHTTP()
}

func clientsLoop() {
	clients = make(map[string]*client)
	messages = make(chan *msg)
	enter = make(chan *client)
	leave = make(chan string)

	for {
		select {
		case m := <-messages:
			recordLog(m)
			if m.Pri {
				if cli, ok := clients[m.From]; ok {
					go sendTo(cli.ch, m)
				}
				if cli, ok := clients[m.To]; ok {
					go sendTo(cli.ch, m)
				}
				continue
			}

			for _, cli := range clients {
				go sendTo(cli.ch, m)
			}

		case cli := <-enter:
			if old, ok := clients[cli.id]; ok {
				close(old.kick)
			}
			clients[cli.id] = cli
			boradcaseUsers(clients)

		case id := <-leave:
			delete(clients, id)
			boradcaseUsers(clients)
		}

	}
}

func boradcaseUsers(m map[string]*client) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	for _, c := range m {
		go func(c *client) {
			c.users <- keys
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
