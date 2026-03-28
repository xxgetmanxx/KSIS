package main

import (
	"fmt"
	"lab_2/CHAT"
)

func main() {

	var ip string

	var port int

	var name string

	fmt.Printf("| ADRR | ")
	fmt.Scanln(&ip)

	fmt.Printf("| PORT | ")
	fmt.Scanln(&port)

	fmt.Printf("| NAME | ")
	fmt.Scanln(&name)

	CHAT.StartUdp(ip, port, name)

	go CHAT.StartTcpServer(ip, port)

	CHAT.RunChat(name)

}
