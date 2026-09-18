package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	conn, err := net.ListenPacket("udp", "127.0.0.1:8001")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	fmt.Println("UDP-сервер: 127.0.0.1:8001")
	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			log.Print(err)
			continue
		}
		fmt.Printf("%s: %s\n", addr, buf[:n])
		if _, err := conn.WriteTo([]byte("Hello, client"), addr); err != nil {
			log.Print(err)
		}
	}
}
