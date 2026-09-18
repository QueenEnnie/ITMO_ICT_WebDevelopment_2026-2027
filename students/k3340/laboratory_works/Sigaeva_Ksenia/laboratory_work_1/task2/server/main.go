package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"net"
	"strconv"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:8002")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	fmt.Println("Сервер запущен: 127.0.0.1:8002")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		result := solve(scanner.Text())
		if _, err := fmt.Fprintln(conn, result); err != nil {
			log.Print(err)
		}
	}
}

func solve(line string) string {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return "Ошибка: введите ровно 3 коэффициента"
	}

	a, errA := strconv.ParseFloat(fields[0], 64)
	b, errB := strconv.ParseFloat(fields[1], 64)
	c, errC := strconv.ParseFloat(fields[2], 64)

	if errA != nil || errB != nil || errC != nil {
		return "Ошибка: введите корректные числа"
	}

	if math.IsNaN(a) || math.IsNaN(b) || math.IsNaN(c) ||
		math.IsInf(a, 0) || math.IsInf(b, 0) || math.IsInf(c, 0) {
		return "Ошибка: коэффициенты должны быть конечными числами"
	}

	if a == 0 {
		if b == 0 {
			if c == 0 {
				return "Бесконечно много решений"
			}
			return "Решений нет"
		}
		return fmt.Sprintf("Линейное уравнение: x = %g", -c/b)
	}

	d := b*b - 4*a*c
	if d < 0 {
		return "Действительных корней нет"
	}
	if d == 0 {
		return fmt.Sprintf("x = %g", -b/(2*a))
	}

	x1 := (-b + math.Sqrt(d)) / (2 * a)
	x2 := (-b - math.Sqrt(d)) / (2 * a)

	return fmt.Sprintf("x1 = %g; x2 = %g", x1, x2)
}
