package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	fmt.Println("Решение ax² + bx + c = 0. Введите a b c через пробел (например: 1 -3 2):")
	input := bufio.NewScanner(os.Stdin)
	if !input.Scan() {
		return
	}
	conn, err := net.DialTimeout("tcp", "127.0.0.1:8002", 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := fmt.Fprintln(conn, input.Text()); err != nil {
		log.Fatal(err)
	}
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(response)
}
