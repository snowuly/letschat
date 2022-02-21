package letschat

import (
	"io"
	"net/http"
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
