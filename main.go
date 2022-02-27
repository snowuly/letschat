package letschat

import "log"

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

func GetRoomList(uid string) (list []*RoomInfo) {
	for _, room := range rooms {
		list = append(list, room.Info(uid))
	}
	return
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
