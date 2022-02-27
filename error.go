package letschat

import "fmt"

type MsgErrno int

var errs = [...]string{
	0: "sender not found",
	1: "receiver not found",
	2: "content is empty",
	3: "user is not online",
}

func (e MsgErrno) Error() string {
	if 0 <= int(e) && int(e) <= len(errs) {
		return errs[e]
	}
	return fmt.Sprintf("msg errno: %d", e)
}
