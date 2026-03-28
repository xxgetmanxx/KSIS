package main

import (
	"fmt"
	"lab_2/CHAT"
)

func main() {

	var ip string

	var port int

	var name string

	fmt.Print("ip: ")

	fmt.Scanln(&ip)

	fmt.Print("port: ")

	fmt.Scanln(&port)

	fmt.Print("name: ")

	fmt.Scanln(&name)

	CHAT.StartUdp(ip, port, name)

	go CHAT.StartTcpServer(ip, port)

	CHAT.RunChat(name)

}
