package handlers

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
	"homeworkboard/internal/auth"
	database "homeworkboard/internal/db"
)

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	p := Page{Title: "Регистрация", Form: map[string]string{"role": "student"}}
	if r.Method == "GET" {
		a.render(w, r, "register", p, 200)
		return
	}
	name := strings.TrimSpace(r.PostForm.Get("name"))
	username := strings.ToLower(strings.TrimSpace(r.PostForm.Get("username")))
	password := r.PostForm.Get("password")
	role := r.PostForm.Get("role")
	p.Form = map[string]string{"name": name, "username": username, "role": role}
	if role != "student" && role != "teacher" {
		p.Error = "Выберите роль: ученик или учитель."
		a.render(w, r, "register", p, 422)
		return
	}
	if role == "teacher" && subtle.ConstantTimeCompare([]byte(r.PostForm.Get("teacher_key")), []byte(auth.TeacherRegistrationKey)) != 1 {
		p.Error = "Для регистрации учителя введите правильный ключ учителя."
		a.render(w, r, "register", p, 422)
		return
	}
	if name == "" || utf8.RuneCountInString(name) > 100 || !auth.ValidUsername(username) || utf8.RuneCountInString(password) < 8 {
		p.Error = "Укажите имя, логин (3–40 латинских букв, цифр или _) и пароль длиной не менее 8 символов."
		a.render(w, r, "register", p, 422)
		return
	}
	if len(password) > 72 {
		p.Error = "Пароль слишком длинный. Выберите более короткий пароль."
		a.render(w, r, "register", p, 422)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		a.fail(w, err)
		return
	}
	err = database.CreateUser(r.Context(), a.db, username, name, string(hash), role)
	if database.Duplicate(err) {
		p.Error = "Этот логин уже занят."
		a.render(w, r, "register", p, 422)
		return
	}
	if err != nil {
		a.fail(w, err)
		return
	}
	http.Redirect(w, r, "/login?registered=1", 303)
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	p := Page{Title: "Вход", Form: map[string]string{}}
	if r.Method == "GET" {
		a.render(w, r, "login", p, 200)
		return
	}
	username := strings.ToLower(strings.TrimSpace(r.PostForm.Get("username")))
	p.Form["username"] = username
	id, hash, err := database.GetCredentials(r.Context(), a.db, username)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		a.fail(w, err)
		return
	}
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(r.PostForm.Get("password"))) != nil {
		p.Error = "Неверный логин или пароль."
		a.render(w, r, "login", p, 422)
		return
	}
	session, err := auth.Token()
	if err != nil {
		a.fail(w, err)
		return
	}
	err = database.CreateSession(r.Context(), a.db, auth.TokenHash(session), id)
	if err != nil {
		a.fail(w, err)
		return
	}
	if old, err := r.Cookie("session"); err == nil {
		_ = database.DeleteSession(r.Context(), a.db, auth.TokenHash(old.Value))
	}
	a.auth.Cookie(w, "session", session, 86400)
	http.Redirect(w, r, "/assignments", 303)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("session"); err == nil {
		if err = database.DeleteSession(r.Context(), a.db, auth.TokenHash(c.Value)); err != nil {
			a.fail(w, err)
			return
		}
	}
	a.auth.Cookie(w, "session", "", -1)
	http.Redirect(w, r, "/login", 303)
}
