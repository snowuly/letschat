package letschat

import (
	"errors"
	"fmt"
)

type credit map[int]map[string]bool

var cre credit

func (c credit) add(roomID int, uid string) {
	if _, ok := c[roomID]; !ok {
		c[roomID] = make(map[string]bool)
	}

	c[roomID][uid] = true
}

func (c credit) remove(roomID int, uid string) {
	delete(cre[roomID], uid)
}

func GetCredit(roomID int, uid, pwd string) error {
	room := GetRoom(roomID)
	if room == nil {
		return fmt.Errorf("room not found: %d", roomID)
	}
	if room.pwd == "" {
		return nil
	}
	if room.pwd != pwd {
		return errors.New("wrong password")
	}
	cre.add(roomID, uid)
	return nil
}

func HasCredit(roomID int, uid string) bool {
	umap, ok := cre[roomID]
	if !ok {
		return false
	}
	return umap[uid]

}
func dropCredit(roomID int, uid string) {
	delete(cre[roomID], uid)
}

func init() {
	cre = make(credit)
}
