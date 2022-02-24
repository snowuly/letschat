package letschat

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	secret2name map[string]string
	admin       string
	chatRoom    *Room
)

type SessionItem struct {
	id     string
	expire time.Time
}

var sessionMu sync.RWMutex
var session map[string]SessionItem

var ErrNotFound = errors.New("Not Found")

func getSessionItem(sid string) (string, error) {
	sessionMu.RLock()
	defer sessionMu.RUnlock()
	if item, ok := session[sid]; ok {
		return item.id, nil
	}

	return "", ErrNotFound
}

func SetSessionItem(sid, id string) {
	sessionMu.Lock()
	session[sid] = SessionItem{id, time.Now().Add(1 * time.Hour)}
	sessionMu.Unlock()
}

func init() {
	session = make(map[string]SessionItem)

	// secret2name = map[string]string{
	// 	"haha": "系统管理员",
	// 	"hehe": "Tony",
	// 	"xixi": "John",
	// }
	var err error
	secret2name, admin, err = getUserMap()
	if err != nil {
		log.Fatal(fmt.Errorf("getUserMap error: %v", err))
	}
}

func StartHTTP() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK, Good Boy~")
	})
	http.HandleFunc("/user", handleUser)
	http.HandleFunc("/login", handleLogin)
	http.HandleFunc("/send", handleSend)
	http.HandleFunc("/clear", handleClearLog)
	http.HandleFunc("/admin", handleAdmin)
	http.HandleFunc("/sse", handleSSE)

	log.Fatal(http.ListenAndServe("127.0.0.1:8900", nil))
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	name := handleName(w, r)

	if name == "" {
		return
	}

	txt := r.PostFormValue("txt")
	to := r.PostFormValue("to")
	pri := r.PostFormValue("pri")

	if txt == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if to != "" {
		if i, _ := chatRoom.getUser(to); i < 0 {
			http.Error(w, to+" is not online", http.StatusBadRequest)
			return
		}
	}

	go chatRoom.Send(name, to, txt, pri != "" && to != "")

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}
func handleUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	name := handleName(w, r)

	if name == "" {
		return
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, name)
}
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	secret := r.PostFormValue("secret")

	if secret == "" {
		http.Error(w, "secret is required", http.StatusForbidden)
		return
	}

	name, ok := secret2name[secret]
	if !ok {
		http.Error(w, "invalid secret", http.StatusForbidden)
		return
	}

	sid := genSid()
	SetSessionItem(sid, name)

	http.SetCookie(w, &http.Cookie{Name: "sid", Path: "/", Value: sid, HttpOnly: true, MaxAge: 3600 * 24})
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, name)
}

func handleLogout() {

}

func handleAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	name := handleName(w, r)

	if name == "" {
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%t", name == admin)
}

func handleClearLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	name := handleName(w, r)

	if name == "" {
		return
	}

	if name != admin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	chatRoom.clearLog()

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	name := handleName(w, r)

	if name == "" {
		return
	}

	w.Header().Set("content-type", "text/event-stream; charset=utf-8")

	ctx := r.Context()

	user := &User{
		name,
		"",
		r.Header.Get("X-Real-IP"),
		name == admin,
		make(chan []byte),
	}

	go chatRoom.Enter(user)

	flushWriteString(w, "event: log\ndata: "+string(chatRoom.getLog(user.ID))+"\n\n")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	idleTicker := time.NewTicker(5 * time.Hour)
	defer idleTicker.Stop()

	for {
		select {
		case m, ok := <-user.ch:
			if !ok {
				flushWriteString(w, "event: close\ndata: \n\n")
				return
			}

			end := len(m) - 1
			if m[end] == 0 { // msg
				flushWriteString(w, "event: msg\ndata: "+string(m[:end])+"\n\n")
			} else if m[end] == 1 {
				flushWriteString(w, "event: users\ndata: "+string(m[:end])+"\n\n")
			}
			idleTicker.Reset(5 * time.Hour)

		case <-ticker.C:
			flushWriteString(w, ": tick\n\n")

		case <-idleTicker.C:
			chatRoom.Leave(name)
			return

		case <-ctx.Done():
			chatRoom.Leave(name)
			return
		}
	}
}

func genSid() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func flushWriteString(w http.ResponseWriter, s string) {
	io.WriteString(w, s)
	w.(http.Flusher).Flush()
}
