package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type client struct {
	name string
	out  chan string
}

var (
	mu      sync.Mutex
	clients = make(map[net.Conn]*client)
)

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:8004")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("Чат: 127.0.0.1:8004")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handle(conn)
	}
}

func broadcast(sender net.Conn, message string) {
	mu.Lock()
	defer mu.Unlock()
	for conn, peer := range clients {
		if conn == sender {
			continue
		}
		select {
		case peer.out <- message:
		default:
			// Медленный клиент отключается, остальные продолжают получать сообщения.
			delete(clients, conn)
			close(peer.out)
			go conn.Close()
		}
	}
}

func writeMessages(conn net.Conn, peer *client) {
	defer conn.Close()
	for message := range peer.out {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if _, err := fmt.Fprintln(conn, message); err != nil {
			return
		}
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}
	name := strings.TrimSpace(scanner.Text())
	if name == "" {
		fmt.Fprintln(conn, "Имя не может быть пустым")
		return
	}
	mu.Lock()
	for _, existing := range clients {
		if existing.name == name {
			mu.Unlock()
			fmt.Fprintln(conn, "Это имя уже занято")
			return
		}
	}
	peer := &client{name: name, out: make(chan string, 64)}
	clients[conn] = peer
	mu.Unlock()
	go writeMessages(conn, peer)
	fmt.Println(name, "подключился")
	broadcast(conn, name+" вошёл в чат")
	for scanner.Scan() {
		message := scanner.Text()
		if message == "exit" {
			break
		}
		if strings.TrimSpace(message) != "" {
			broadcast(conn, name+": "+message)
		}
	}
	mu.Lock()
	if _, ok := clients[conn]; ok {
		delete(clients, conn)
		close(peer.out)
	}
	mu.Unlock()
	broadcast(conn, name+" вышел из чата")
	fmt.Println(name, "отключился")
}
