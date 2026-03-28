package CHAT

import (
	"bufio"
	"fmt"
	"os"
)

func RunChat(name string) {

	reader := bufio.NewReader(os.Stdin)

	for {

		fmt.Print("| TEXT | ")
		text, _ := reader.ReadString('\n')

		msg := makeMsg(msgText, name+": "+text)

		for _, c := range peers {

			c.Write(msg)

		}

	}

}
