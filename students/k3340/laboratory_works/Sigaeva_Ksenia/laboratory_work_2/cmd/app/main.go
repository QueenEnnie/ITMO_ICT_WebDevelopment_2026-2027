package main

import (
	"log"
	"net/http"
	"os"
	"time"

	database "homeworkboard/internal/db"
	"homeworkboard/internal/handlers"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("Задайте DATABASE_URL: postgres://user:password@127.0.0.1:5432/homework?sslmode=disable")
	}
	conn, err := database.Open(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	app, err := handlers.New(conn, os.Getenv("COOKIE_SECURE") == "true")
	if err != nil {
		log.Fatal(err)
	}
	addr := os.Getenv("APP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	server := &http.Server{
		Addr: addr, Handler: app.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("Доска заданий: http://%s", addr)
	log.Fatal(server.ListenAndServe())
}
