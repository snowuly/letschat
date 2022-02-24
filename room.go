package letschat

import (
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

type Msg struct {
	ID, From, To, Txt string
	Time              int64
	Priv              bool
}

type User struct {
	ID, Name, IP string
	Admin        bool
	ch           chan []byte // json + TYPE(0: message 1: users)
}

type Room struct {
	ID    int
	Name  string
	Users []*User
	msg   chan *Msg
	enter chan *User
	leave chan string
	log   *list.List
	file  *os.File
	sync.RWMutex
}

func (r *Room) Start() error {
	file, err := os.OpenFile(fmt.Sprintf("r-%d-log", r.ID), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer file.Close()
	r.file = file

	msgPrefix := []byte("[msg]")

	for {
		select {
		case m := <-r.msg:
			r.addLog(m)
			b, _ := json.Marshal(m)
			r.file.Write(msgPrefix)
			r.file.Write(append(b, '\n'))
			data := append(b, 0)

			if m.Priv {
				for _, user := range r.Users {
					if user.ID == m.To || user.ID == m.From {
						go send(user.ch, data)
					}
				}
				continue
			}

			for _, user := range r.Users {
				go send(user.ch, data)
			}
		case user := <-r.enter:
			r.file.WriteString(fmt.Sprintf("[user]: %s %s %s enter\n", user.ID, user.Name, user.IP))
			if i, old := r.getUser(user.ID); i >= 0 {
				r.Users[i] = user
				close(old.ch)
			} else {
				r.Users = append(r.Users, user)
			}
			r.broadcastUsers()
		case id := <-r.leave:
			user := r.removeUser(id)
			r.file.WriteString(fmt.Sprintf("[user]: %s %s %s leave\n", user.ID, user.Name, user.IP))
			r.broadcastUsers()
		}
	}
}

func (r *Room) getUser(id string) (int, *User) {
	for index, user := range r.Users {
		if user.ID == id {
			return index, user
		}
	}
	return -1, nil
}

func (r *Room) removeUser(id string) (user *User) {
	index := -1

	for i, u := range r.Users {
		if u.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}

	user = r.Users[index]
	r.Users = append(r.Users[:index], r.Users[index+1:]...)

	return user
}

func (r *Room) addLog(m *Msg) {
	r.Lock()
	r.log.PushBack(m)
	if r.log.Len() > 200 {
		r.log.Remove(r.log.Front())
	}
	r.Unlock()
}

func (r *Room) getLog(id string) []byte {
	r.RLock()
	defer r.RUnlock()

	list := make([]*Msg, 0, r.log.Len())

	for e := r.log.Front(); e != nil; e = e.Next() {
		m := e.Value.(*Msg)
		if !m.Priv || m.From == id || m.To == id {
			list = append(list, m)
		}
	}

	b, _ := json.Marshal(list)

	return b
}

func (r *Room) clearLog() {
	r.Lock()
	r.log.Init()
	r.Unlock()
}

func (r *Room) broadcastUsers() {
	b, _ := json.Marshal(r.Users)
	data := append(b, 1)

	for _, u := range r.Users {
		go func(u *User) {
			u.ch <- data
		}(u)
	}
}

func (r *Room) Send(from, to, txt string, priv bool) {
	r.msg <- &Msg{
		<-mid,
		from,
		to,
		txt,
		time.Now().Unix(),
		priv,
	}
}

func (r *Room) Enter(u *User) {
	r.enter <- u
}

func (r *Room) Leave(id string) {
	r.leave <- id
}

func NewRoom(name string) *Room {
	return &Room{
		ID:    <-rid,
		Name:  name,
		Users: make([]*User, 0),
		msg:   make(chan *Msg, 10),
		enter: make(chan *User),
		leave: make(chan string),
		log:   list.New(),
	}
}

var (
	rid chan int
	mid chan string
)

func init() {
	rid = make(chan int)
	mid = make(chan string)

	go func() {
		var i int
		var j int64

		ts := strconv.FormatInt(time.Now().Unix(), 36)

		for {
			select {
			case rid <- i:
				i++
			case mid <- fmt.Sprintf("%s-%d", ts, j):
				j++
			}
		}
	}()

}

func send(ch chan<- []byte, s []byte) {
	select {
	case ch <- s:
	case <-time.After(5 * time.Second):
		log.Printf("send msg timeout: %s", s)
	}
}
