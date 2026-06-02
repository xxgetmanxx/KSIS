package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
)

var blacklist = make(map[string]bool)

func main() {

	loadBlacklist("blacklist.txt")

	listener, err := net.Listen("tcp", "127.0.0.2:8080")

	if err != nil {

		log.Fatalf("Ошибка запуска сервера")

	}

	defer listener.Close()

	fmt.Println("Прокси-сервер запущен")

	for {

		clientConn, err := listener.Accept()

		if err != nil {

			log.Println("Ошибка подключения клиента")

			continue

		}

		go handleConnection(clientConn)

	}

}

func loadBlacklist(filename string) {

	file, err := os.Open(filename)

	if err != nil {

		log.Println("Файл не найден")

		return

	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {

		domain := strings.TrimSpace(scanner.Text())

		if domain != "" {

			blacklist[domain] = true

		}

	}

}

func handleConnection(clientConn net.Conn) {

	defer clientConn.Close()

	

	clientReader := bufio.NewReader(clientConn)

	reqLine, err := clientReader.ReadString('\n')

	if err != nil {

		return

	}

	reqLine = strings.TrimSpace(reqLine)

	

	parts := strings.Split(reqLine, " ")

	if len(parts) != 3 {

		return

	}

	method, rawURL, proto := parts[0], parts[1], parts[2]

	if method == "CONNECT" {

		clientConn.Write([]byte("HTTP/1.1 501 Not Implemented\r\n\r\n"))

		return

	}

	

	parsedURL, err := url.Parse(rawURL)

	if err != nil {

		return

	}

	host := parsedURL.Hostname()

	port := parsedURL.Port()

	if port == "" {

		port = "80"

	}

	targetAddress := host + ":" + port

	

	if blacklist[host] {

		fmt.Printf("%s - BLOCKED\n", host)

		sendErrorPage(clientConn, host)

		return

	}

	

	path := parsedURL.Path

	if parsedURL.RawQuery != "" {

		path += "?" + parsedURL.RawQuery

	}

	if path == "" {

		path = "/"

	}

	

	targetConn, err := net.Dial("tcp", targetAddress)

	if err != nil {

		log.Println("Не удалось подключиться")

		return

	}

	defer targetConn.Close()

	

	newReqLine := fmt.Sprintf("%s %s %s\r\n", method, path, proto)

	targetConn.Write([]byte(newReqLine))

	

	go io.Copy(targetConn, clientReader)

	

	targetReader := bufio.NewReader(targetConn)

	respLine, err := targetReader.ReadString('\n')

	if err != nil {

		return

	}

	statusText := "Unknown"

	respParts := strings.SplitN(strings.TrimSpace(respLine), " ", 3)

	if len(respParts) >= 3 {

		statusText = respParts[1] + " " + respParts[2]

	} else if len(respParts) == 2 {

		statusText = respParts[1]

	}

	fmt.Printf("%s - %s\n", rawURL, statusText)

	clientConn.Write([]byte(respLine))

	

	io.Copy(clientConn, targetReader)

}

func sendErrorPage(conn net.Conn, host string) {

	html := fmt.Sprintf(`<html>
<head><title>Access Denied</title></head>
<body>
	<h1 style="color:red;">Доступ запрещен</h1>
	<p>Ресурс <b>%s</b> находится в черном списке прокси-сервера.</p>
</body>
</html>`, host)

	resp := fmt.Sprintf("HTTP/1.1 403 Forbidden\r\n"+
		"Content-Type: text/html; charset=utf-8\r\n"+
		"Content-Length: %d\r\n"+
		"Connection: close\r\n\r\n"+
		"%s", len(html), html)

	conn.Write([]byte(resp))

}
