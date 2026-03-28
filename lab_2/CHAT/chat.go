package CHAT

import (
	"bufio"
	"os"
)

func RunChat(name string) {

	reader := bufio.NewReader(os.Stdin)

	for {

		text, _ := reader.ReadString('\n')

		msg := makeMsg(msgText, name+": "+text)

		for _, c := range peers {

			c.Write(msg)

		}

	}

}
