package letschat

import (
	"io"
	"net/http"
	"strconv"
)

func handleName(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("sid")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "Need Login")
		return ""
	}
	name, err := getSessionItem(cookie.Value)

	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		cookie.MaxAge = -1
		http.SetCookie(w, cookie)
		io.WriteString(w, "Forbidden")
		return ""
	}
	return name
}

var (
	MSG_PREFIX   = []byte("event: msg\ndata: ")
	USERS_PREFIX = []byte("event: users\ndata: ")
	LOG_PREFIX   = []byte("event: log\ndata: ")
)

func flushWrite(w http.ResponseWriter, kind byte, data []byte) {
	var output []byte
	switch kind {
	case 0:
		output = append(MSG_PREFIX, data...)
	case 1:
		output = append(USERS_PREFIX, data...)
	case 2:
		output = append(LOG_PREFIX, data...)
	default:
		output = []byte{':'}
	}
	output = append(output, '\n', '\n')
	w.Write(output)
	w.(http.Flusher).Flush()
}

func handleRoomReq(w http.ResponseWriter, r *http.Request) *Room {
	room := r.URL.Query().Get("room")
	if room == "" {
		if len(rooms) > 0 {
			http.Error(w, "room is required", http.StatusBadRequest)
			return nil
		} else {
			return defaultRoom
		}
	}

	id, err := strconv.Atoi(room)
	if err != nil {
		http.Error(w, "room err: "+err.Error(), http.StatusBadRequest)
		return nil
	}

	ins := GetRoom(id)

	if ins == nil {
		http.Error(w, "room not found", http.StatusBadRequest)
		return nil
	}

	return ins
}
