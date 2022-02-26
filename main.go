package letschat

import "log"

type RoomInfo struct {
	ID     int
	Name   string
	Num    int
	Priv   bool
	Credit bool
}

const max = 20

var (
	rooms       []*Room
	defaultRoom *Room
)

func init() {
	rooms = make([]*Room, 0, max)
}

func AddRoom(name, pwd string) {
	if len(rooms) >= max {
		log.Fatal("AddRoom: too many rooms")
	}
	rooms = append(rooms, NewRoom(name, pwd))
}

func GetRoom(id int) *Room {
	for _, r := range rooms {
		if r.ID == id {
			return r
		}
	}

	return nil
}

func GetRoomList(uid string) (list []*RoomInfo) {
	for _, room := range rooms {
		list = append(list, getRoomInfo(room, uid))
	}
	return
}

func getRoomInfo(room *Room, uid string) *RoomInfo {
	priv := room.pwd != ""
	var hasCredit bool
	var num int

	if priv {
		hasCredit = HasCredit(room.ID, uid)
	}

	if !priv || hasCredit {
		num = len(room.Users)
	}

	if !priv || HasCredit(room.ID, uid) {
		num = len(room.Users)
	}

	return &RoomInfo{
		room.ID,
		room.Name,
		num,
		priv,
		hasCredit,
	}
}

func GetRoomInfo(uid string, rid int) *RoomInfo {
	room := GetRoom(rid)
	if room == nil {
		return nil
	}

	return getRoomInfo(room, uid)
}

func Start() {
	if len(rooms) == 0 {
		defaultRoom = NewRoom("默认房间", "")
		go defaultRoom.Start()
	} else {
		for _, room := range rooms {
			go room.Start()
		}
	}

	StartHTTP()
}
