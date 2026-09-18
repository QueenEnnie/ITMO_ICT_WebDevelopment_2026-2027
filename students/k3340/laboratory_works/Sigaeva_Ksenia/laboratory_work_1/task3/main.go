package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	page, err := os.ReadFile("task3/index.html")
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:8003")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("Откройте http://127.0.0.1:8003")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		serve(conn, page)
	}
}

func serve(conn net.Conn, page []byte) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}
	_, err := fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(page), page)
	if err != nil {
		log.Print(err)
	}
}
