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
	addr := "127.0.0.1:5000"

	ln, _ := net.Listen("tcp", addr)
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

			sendHistory(conn)

			clients = append(clients, Client{Name: name, Conn: conn})
			addMessage("", name+" (JOIN)", true)
			broadcast(fmt.Sprintf("[%s] <SYSTEM> %s (JOIN)", getTime(), name))

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

	fmt.Printf("[ADDRESS] %s\n", addr)
	fmt.Printf("[CLIENTS] %d\n", len(clients))
	fmt.Printf("[HISTORY] %d\n", len(history))

	for {
		time.Sleep(1 * time.Second)
		fmt.Printf("\033[3A\033[K")
		fmt.Printf("[ADDRESS] %s\n\033[K", addr)
		fmt.Printf("[CLIENTS] %d\n\033[K", len(clients))
		fmt.Printf("[HISTORY] %d\n\033[K", len(history))
	}
}

func runClient() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("[SERV] ")
	serv, _ := reader.ReadString('\n')
	serv = strings.TrimSpace(serv)

	fmt.Print("[ADDR] ")
	addr, _ := reader.ReadString('\n')
	addr = strings.TrimSpace(addr)

	fmt.Print("[NAME] ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	ln, _ := net.Listen("tcp", addr)
	defer ln.Close()

	conn, _ := net.Dial("tcp", serv)
	defer conn.Close()

	conn.Write([]byte(name + "\n"))

	done := make(chan bool)

	go func() {
		r := bufio.NewReader(os.Stdin)
		for {
			msg, _ := r.ReadString('\n')
			msg = strings.TrimSpace(msg)
			if msg != "" {
				conn.Write([]byte(msg + "\n"))
			}
		}
	}()

	go func() {
		r := bufio.NewReader(conn)
		for {
			msg, err := r.ReadString('\n')
			if err != nil {
				done <- true
				return
			}
			fmt.Println(strings.TrimSpace(msg))
		}
	}()

	<-done
}

func main() {
	fmt.Println("[SERVER] 0")
	fmt.Println("[CLIENT] 1")
	
	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "0" {
		runServer()
	} else {
		runClient()
	}
}
