package letschat

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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
	http.HandleFunc("/rooms", handleRooms)
	http.HandleFunc("/room", handleRoom)
	http.HandleFunc("/roompwd", handleRoomPwd)
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
	priv := r.PostFormValue("priv")
	token := r.PostFormValue("token")

	if txt == "" || token == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	room := handleRoomReq(w, r)

	if room == nil {
		return
	}

	if err := <-room.Send(token, name, to, txt, priv != ""); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

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
	fmt.Fprintf(w, "%s|%t", name, name == admin)
}
func handleRooms(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	name := handleName(w, r)

	if name == "" {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GetRoomList(name))
}

func handleRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	name := handleName(w, r)
	if name == "" {
		return
	}

	room := handleRoomReq(w, r)
	if room == nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(room.Info(name))
}

func handleRoomPwd(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	name := handleName(w, r)
	if name == "" {
		return
	}

	pwd := r.PostFormValue("pwd")
	if pwd == "" {
		http.Error(w, "password is empty", http.StatusBadRequest)
		return
	}

	room := handleRoomReq(w, r)
	if room == nil {
		return
	}

	if err := GetCredit(room.ID, name, pwd); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Write([]byte("OK"))
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
	fmt.Fprintf(w, "%s|%t", name, name == admin)
}

func handleLogout() {

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

	room := handleRoomReq(w, r)

	if room == nil {
		return
	}

	room.ClearLog()

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

	token := fmt.Sprintf("%d", time.Now().UnixMilli()) // TODO: shouldn't be predictable

	room := handleRoomReq(w, r)
	if room == nil {
		return
	}

	if room.pwd != "" {
		if !HasCredit(room.ID, name) {
			w.Write([]byte("event: err\ndata: pwd\n\n"))
			return
		}
	}

	ch, token := room.Enter(name, "", r.Header.Get("X-Real-IP"), name == admin)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	idleTicker := time.NewTicker(5 * time.Hour)
	defer idleTicker.Stop()

	io.WriteString(w, fmt.Sprintf("event: token\ndata: %s\n\n", token))
	w.(http.Flusher).Flush()

	for {
		select {
		case m, ok := <-ch:
			if !ok {
				w.Write([]byte("event: err\ndata: kicked\n\n"))
				return
			}

			end := len(m) - 1
			flushWrite(w, m[end], m[:end])

		case <-ticker.C:
			w.Write([]byte(": tick\n\n"))
			w.(http.Flusher).Flush()

		case <-idleTicker.C:
			room.Leave(name)
			return

		case <-ctx.Done():
			room.Leave(name)
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
