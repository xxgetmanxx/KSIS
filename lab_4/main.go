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

	listener, err := net.Listen("tcp", ":8080")

	if err != nil {

		log.Fatalf("Ошибка запуска сервера")

	}

	defer listener.Close()

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
	// Гарантированно закрываем соединение с клиентом при выходе из функции
	defer clientConn.Close()

	clientReader := bufio.NewReader(clientConn)

	// Шаг 1: Читаем первую строку HTTP-запроса (Request Line)
	reqLine, err := clientReader.ReadString('\n')
	if err != nil {
		return
	}
	reqLine = strings.TrimSpace(reqLine)

	// Разбиваем строку на Метод, URL и Версию протокола (например: GET http://example.com/ HTTP/1.1)
	parts := strings.Split(reqLine, " ")
	if len(parts) != 3 {
		return
	}
	method, rawURL, proto := parts[0], parts[1], parts[2]

	// Отклоняем HTTPS (CONNECT), так как по заданию поддержка не требуется
	if method == "CONNECT" {
		clientConn.Write([]byte("HTTP/1.1 501 Not Implemented\r\n\r\n"))
		return
	}

	// Шаг 2: Парсим URL, чтобы достать хост и путь
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return
	}

	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "80" // По умолчанию для HTTP
	}
	targetAddress := host + ":" + port

	// Шаг 3: Проверка на черный список (Доп. задание)
	if blacklist[host] {
		log.Printf("[BLOCKED] URL: %s", rawURL)
		sendErrorPage(clientConn, host)
		return
	}

	// Шаг 4: Формируем правильный URI (переделываем абсолютный URL в путь)
	path := parsedURL.Path
	if parsedURL.RawQuery != "" {
		path += "?" + parsedURL.RawQuery
	}
	if path == "" {
		path = "/"
	}

	// Шаг 5: Устанавливаем сокет-соединение с целевым сервером
	targetConn, err := net.Dial("tcp", targetAddress)
	if err != nil {
		log.Printf("Не удалось подключиться к %s: %v", targetAddress, err)
		return
	}
	defer targetConn.Close()

	// Шаг 6: Отправляем целевому серверу модифицированную строку запроса
	newReqLine := fmt.Sprintf("%s %s %s\r\n", method, path, proto)
	targetConn.Write([]byte(newReqLine))

	// Шаг 7: Асинхронно пересылаем оставшиеся заголовки и тело запроса от браузера к серверу
	go io.Copy(targetConn, clientReader)

	// Шаг 8: Читаем ответ от целевого сервера, чтобы залогировать статус-код
	targetReader := bufio.NewReader(targetConn)
	respLine, err := targetReader.ReadString('\n')
	if err != nil {
		return
	}

	// Парсим статус-код из ответа (например: HTTP/1.1 200 OK)
	statusCode := "Unknown"
	respParts := strings.SplitN(strings.TrimSpace(respLine), " ", 3)
	if len(respParts) >= 2 {
		statusCode = respParts[1]
	}

	// Журналирование проксируемых запросов по заданию
	log.Printf("[PROXY] URL: %s | Code: %s\n", rawURL, statusCode)

	// Отправляем первую строку ответа обратно клиенту
	clientConn.Write([]byte(respLine))

	// Шаг 9: Синхронно пересылаем оставшуюся часть ответа (заголовки и тело/аудиопоток) клиенту
	io.Copy(clientConn, targetReader)
}

// Функция для отправки кастомной страницы ошибки
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
