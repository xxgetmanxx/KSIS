package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	MsgText   = 0x01
	MsgSystem = 0x02
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
var stopServer = make(chan bool)

func getTime() string {
	return time.Now().Format("15:04:05")
}

func addMessage(sender, text string, isSystem bool) {
	history = append(history, Message{Time: getTime(), Sender: sender, Text: text, IsSystem: isSystem})
}

func writeMessage(conn net.Conn, msgType byte, data string) {
	length := len(data)
	if length > 255 {
		length = 255
	}
	conn.Write([]byte{msgType, byte(length)})
	conn.Write([]byte(data[:length]))
}

func readMessage(conn net.Conn) (byte, string, error) {
	header := make([]byte, 2)
	_, err := conn.Read(header)
	if err != nil {
		return 0, "", err
	}
	msgType := header[0]
	length := header[1]
	if length == 0 {
		return msgType, "", nil
	}
	data := make([]byte, length)
	_, err = conn.Read(data)
	if err != nil {
		return msgType, "", err
	}
	return msgType, string(data), nil
}

func broadcast(msgType byte, data string) {
	for _, c := range clients {
		writeMessage(c.Conn, msgType, data)
	}
}

func sendHistory(conn net.Conn) {
	for _, m := range history {
		var data string
		if m.IsSystem {
			data = fmt.Sprintf("[%s] <SYSTEM> %s", m.Time, m.Text)
		} else {
			data = fmt.Sprintf("[%s] <%s> %s", m.Time, m.Sender, m.Text)
		}
		writeMessage(conn, MsgSystem, data)
	}
}

func runServer() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("[ADDRESS] ")
	addr, _ := reader.ReadString('\n')
	addr = strings.TrimSpace(addr)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("Error: port %s is already in use\n", addr)
		os.Exit(1)
	}
	defer ln.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		for _, c := range clients {
			c.Conn.Close()
		}
		stopServer <- true
	}()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			_, name, _ := readMessage(conn)

			sendHistory(conn)

			clients = append(clients, Client{Name: name, Conn: conn})
			addMessage("", name+" (JOIN)", true)
			broadcast(MsgSystem, fmt.Sprintf("[%s] <SYSTEM> %s (JOIN)", getTime(), name))

			go func(c Client) {
				defer func() {
					c.Conn.Close()
					for i, cl := range clients {
						if cl.Name == c.Name {
							clients = append(clients[:i], clients[i+1:]...)
							break
						}
					}
					addMessage("", c.Name+" (LEFT)", true)
					broadcast(MsgSystem, fmt.Sprintf("[%s] <SYSTEM> %s (LEFT)", getTime(), c.Name))
				}()
				for {
					msgType, data, err := readMessage(conn)
					if err != nil {
						return
					}
					if msgType == MsgText && data != "" {
						addMessage(c.Name, data, false)
						broadcast(MsgText, fmt.Sprintf("[%s] <%s> %s", getTime(), c.Name, data))
					}
				}
			}(clients[len(clients)-1])
		}
	}()

	fmt.Printf("[CLIENTS] %d\n", len(clients))
	fmt.Printf("[HISTORY] %d\n", len(history))

	for {
		select {
		case <-stopServer:
			fmt.Println("\n[SERVER] Shutdown complete")
			return
		case <-time.After(1 * time.Second):
			fmt.Printf("\033[2A\033[K")
			fmt.Printf("[CLIENTS] %d\n\033[K", len(clients))
			fmt.Printf("[HISTORY] %d\n\033[K", len(history))
		}
	}
}

func runClient() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("[ADDRESS] ")
	addr, _ := reader.ReadString('\n')
	addr = strings.TrimSpace(addr)

	fmt.Print("[NAME] ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	conn, _ := net.Dial("tcp", addr)

	header := make([]byte, 2)
	header[0] = MsgText
	header[1] = byte(len(name))
	conn.Write(header)
	conn.Write([]byte(name))

	done := make(chan bool)

	go func() {
		r := bufio.NewReader(os.Stdin)
		for {
			msg, _ := r.ReadString('\n')
			msg = strings.TrimSpace(msg)
			if msg != "" {
				writeMessage(conn, MsgText, msg)
			}
		}
	}()

	go func() {
		for {
			_, data, err := readMessage(conn)
			if err != nil {
				done <- true
				return
			}
			fmt.Println(data)
		}
	}()

	<-done
	conn.Close()
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
