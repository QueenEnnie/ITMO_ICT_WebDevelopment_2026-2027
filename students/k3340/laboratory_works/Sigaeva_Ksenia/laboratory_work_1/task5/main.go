package main

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	mu     sync.RWMutex
	grades = make(map[string][]int)
)

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:8005")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("Журнал: http://127.0.0.1:8005")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handle(conn)
	}
}

func reply(conn net.Conn, status, body string) {
	_, err := fmt.Fprintf(conn, "HTTP/1.1 %s\r\nContent-Type: text/html; charset=utf-8\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", status, len([]byte(body)), body)
	if err != nil {
		log.Print(err)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	parts := strings.Fields(line)
	if len(parts) != 3 {
		reply(conn, "400 Bad Request", "Некорректный запрос")
		return
	}
	method, path := parts[0], parts[1]
	headers := make(map[string]string)
	for {
		line, err = reader.ReadString('\n')
		if err != nil {
			reply(conn, "400 Bad Request", "Неполные заголовки")
			return
		}
		if line == "\r\n" {
			break
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			reply(conn, "400 Bad Request", "Некорректный заголовок")
			return
		}
		headers[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	if path != "/" {
		reply(conn, "404 Not Found", "Страница не найдена")
		return
	}
	switch method {
	case "GET":
		reply(conn, "200 OK", page())
	case "POST":
		if headers["transfer-encoding"] != "" {
			reply(conn, "400 Bad Request", "Используйте Content-Length")
			return
		}
		if !strings.HasPrefix(headers["content-type"], "application/x-www-form-urlencoded") {
			reply(conn, "415 Unsupported Media Type", "Нужна HTML-форма")
			return
		}
		length, err := strconv.Atoi(headers["content-length"])
		if err != nil || length < 0 || length > 8192 {
			reply(conn, "400 Bad Request", "Некорректная длина тела")
			return
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(reader, body); err != nil {
			reply(conn, "400 Bad Request", "Неполное тело запроса")
			return
		}
		form, err := url.ParseQuery(string(body))
		if err != nil {
			reply(conn, "400 Bad Request", "Некорректная форма")
			return
		}
		subject := strings.TrimSpace(form.Get("subject"))
		grade, err := strconv.Atoi(form.Get("grade"))
		if subject == "" || err != nil || grade < 2 || grade > 5 {
			reply(conn, "400 Bad Request", "Введите дисциплину и оценку от 2 до 5")
			return
		}
		mu.Lock()
		grades[subject] = append(grades[subject], grade)
		mu.Unlock()
		// После формы браузер делает GET, обновление страницы не повторяет POST.
		fmt.Fprint(conn, "HTTP/1.1 303 See Other\r\nLocation: /\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	default:
		reply(conn, "405 Method Not Allowed", "Поддерживаются GET и POST")
	}
}

func page() string {
	mu.RLock()
	snapshot := make(map[string][]int, len(grades))
	for subject, values := range grades {
		snapshot[subject] = append([]int(nil), values...)
	}
	mu.RUnlock()
	var result strings.Builder
	result.WriteString(`<!doctype html>
<html lang="ru">
<head>
    <meta charset="utf-8">
    <title>Журнал оценок</title>
</head>
<body>
    <h1>Журнал оценок</h1>
    <form method="post" action="/">
        <label>
            Дисциплина
            <input name="subject" required>
        </label>
        <label>
            Оценка
            <input name="grade" type="number" min="2" max="5" required>
        </label>
        <button>Добавить</button>
    </form>
    <table border="1">
        <tr>
            <th>Дисциплина</th>
            <th>Оценки</th>
        </tr>
`)

	subjects := make([]string, 0, len(snapshot))
	for subject := range snapshot {
		subjects = append(subjects, subject)
	}
	sort.Strings(subjects)
	for _, subject := range subjects {
		values := make([]string, len(snapshot[subject]))
		for i, grade := range snapshot[subject] {
			values[i] = strconv.Itoa(grade)
		}
		fmt.Fprintf(&result, `        <tr>
            <td>%s</td>
            <td>%s</td>
        </tr>
`, html.EscapeString(subject), strings.Join(values, ", "))
	}
	result.WriteString(`    </table>
</body>
</html>
`)
	return result.String()
}
