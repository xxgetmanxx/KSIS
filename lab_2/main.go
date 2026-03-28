package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

type Message struct {
	Time     string
	Sender   string
	Text     string
	IsSystem bool
}

type Client struct {
	Name string
	Conn net.Conn
}

var history []Message
var clients []Client

func getTime() string {
	return time.Now().Format("15:04:05")
}

func addMessage(sender, text string, isSystem bool) {
	history = append(history, Message{Time: getTime(), Sender: sender, Text: text, IsSystem: isSystem})
}

func broadcast(msg string) {
	for _, c := range clients {
		c.Conn.Write([]byte(msg + "\n"))
	}
}

func sendHistory(conn net.Conn) {
	for _, m := range history {
		if m.IsSystem {
			conn.Write([]byte(fmt.Sprintf("[%s] <SYSTEM> %s\n", m.Time, m.Text)))
		} else {
			conn.Write([]byte(fmt.Sprintf("[%s] <%s> %s\n", m.Time, m.Sender, m.Text)))
		}
	}
}

func runServer() {
	fmt.Print("[ADDRESS] ")
	var addr string
	fmt.Scanln(&addr)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("[ERROR]", err)
		return
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			reader := bufio.NewReader(conn)
			name, _ := reader.ReadString('\n')
			name = strings.TrimSpace(name)

			clients = append(clients, Client{Name: name, Conn: conn})
			addMessage("", name+" (JOIN)", true)
			broadcast(fmt.Sprintf("[%s] <SYSTEM> %s (JOIN)", getTime(), name))

			sendHistory(conn)

			go func(c Client) {
				defer func() {
					for i, cl := range clients {
						if cl.Name == c.Name {
							clients = append(clients[:i], clients[i+1:]...)
							break
						}
					}
					addMessage("", c.Name+" (LEFT)", true)
					broadcast(fmt.Sprintf("[%s] <SYSTEM> %s (LEFT)", getTime(), c.Name))
				}()
				for {
					msg, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					msg = strings.TrimSpace(msg)
					if msg != "" {
						addMessage(c.Name, msg, false)
						broadcast(fmt.Sprintf("[%s] <%s> %s", getTime(), c.Name, msg))
					}
				}
			}(clients[len(clients)-1])
		}
	}()

	for {
		time.Sleep(1 * time.Second)
		fmt.Printf("\r[SERVER] [ADDRESS] %s [CLIENTS] %d [HISTORY] %d", addr, len(clients), len(history))
	}
}

func runClient() {
	fmt.Print("[ADDRESS] ")
	var addr string
	fmt.Scanln(&addr)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("[ERROR]", err)
		return
	}
	defer conn.Close()

	fmt.Print("[NAME] ")
	var name string
	fmt.Scanln(&name)
	conn.Write([]byte(name + "\n"))

	done := make(chan bool)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			msg, _ := reader.ReadString('\n')
			msg = strings.TrimSpace(msg)
			if msg != "" {
				conn.Write([]byte(msg + "\n"))
			}
		}
	}()

	go func() {
		reader := bufio.NewReader(conn)
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				done <- true
				return
			}
			fmt.Println(msg)
		}
	}()

	<-done
}

func main() {
	fmt.Println("[SERVER] 0")
	fmt.Println("[CLIENT] 1")
	var choice int
	fmt.Scan(&choice)

	if choice == 0 {
		runServer()
	} else {
		runClient()
	}
}
