package letschat

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

var userFile = flag.String("u", "", "user file path")

func getUserMap() (users map[string]string, admin string, err error) {
	flag.Parse()

	userMap := make(map[string]string)

	if *userFile == "" {
		err = fmt.Errorf("use -u to specify user file path")
		return
	}

	file, fileErr := os.Open(*userFile)

	if fileErr != nil {
		err = fileErr
		return
	}

	buf := bufio.NewReader(file)

	for {
		line, err := buf.ReadSlice('\n')
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		line = bytes.TrimSpace(line)
		n := bytes.IndexByte(line, ' ')
		if n > 0 {
			name := string(bytes.TrimSpace(line[n+1:]))
			userMap[string(line[:n])] = name

			if admin == "" {
				admin = name
			}
		}

		if err == io.EOF {
			break
		}

	}

	users = userMap

	return
}
