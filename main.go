package letschat

func Start() {
	chatRoom = NewRoom("My Room")

	go chatRoom.Start()

	StartHTTP()
}
