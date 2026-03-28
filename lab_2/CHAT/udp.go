package CHAT

import (
	"fmt"
	"net"
)

func StartUdp(ip string, port int, name string) {

	go listenUdp()

	addr, _ := net.ResolveUDPAddr("udp", "255.255.255.255:19999")

	conn, _ := net.DialUDP("udp", nil, addr)

	msg := makeMsg(msgJoin, fmt.Sprintf("%s|%s|%d", name, ip, port))

	conn.Write(msg)

}

func listenUdp() {

	addr, _ := net.ResolveUDPAddr("udp", ":19999")

	conn, _ := net.ListenUDP("udp", addr)

	buf := make([]byte, 1024)

	for {

		n, _, _ := conn.ReadFromUDP(buf)

		t, data := readMsg(buf[:n])

		if t == msgJoin {

			var name, ip string

			var port int

			fmt.Sscanf(data, "%[^|]|%[^|]|%d", &name, &ip, &port)

			connectTo(ip, port)

		}

	}

}
