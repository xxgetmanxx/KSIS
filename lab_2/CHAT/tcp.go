package CHAT

import (
	"fmt"
	"net"
)

var peers = make(map[string]net.Conn)

func StartTcpServer(ip string, port int) {

	ln, _ := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))

	for {

		conn, _ := ln.Accept()

		go handleConn(conn)

	}

}

func connectTo(ip string, port int) {

	key := fmt.Sprintf("%s:%d", ip, port)

	if _, ok := peers[key]; ok {

		return

	}

	conn, err := net.Dial("tcp", key)

	if err != nil {

		return

	}

	peers[key] = conn

	go handleConn(conn)

}

func handleConn(conn net.Conn) {

	buf := make([]byte, 1024)

	for {

		n, err := conn.Read(buf)

		if err != nil {

			return

		}

		t, text := readMsg(buf[:n])

		if t == msgText {

			fmt.Println(text)

		}

	}

}
