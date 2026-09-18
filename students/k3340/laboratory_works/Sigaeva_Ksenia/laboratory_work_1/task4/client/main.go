package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	input := bufio.NewScanner(os.Stdin)
	fmt.Print("Имя: ")
	if !input.Scan() {
		return
	}
	name := strings.TrimSpace(input.Text())
	if name == "" {
		log.Fatal("Имя не может быть пустым")
	}
	conn, err := net.Dial("tcp", "127.0.0.1:8004")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	if _, err := fmt.Fprintln(conn, name); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Введите сообщение. Для выхода: exit")
	go func() {
		defer conn.Close()
		for input.Scan() {
			message := input.Text()
			if _, err := fmt.Fprintln(conn, message); err != nil {
				return
			}
			if message == "exit" {
				return
			}
		}
	}()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	fmt.Println("Соединение закрыто")
}
