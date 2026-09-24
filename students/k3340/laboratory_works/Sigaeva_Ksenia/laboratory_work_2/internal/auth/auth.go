package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"time"

	database "homeworkboard/internal/db"
	"homeworkboard/internal/models"
)

type Manager struct {
	DB            *sql.DB
	SecureCookies bool
}

func ValidUsername(name string) bool { 
	return models.ValidUsername(name) 
}

func CSRF(r *http.Request) string {
	token, _ := r.Context().Value(csrfKey).(string)
	return token
}

func (a *Manager) fail(w http.ResponseWriter, err error) {
	log.Print(err)
	http.Error(w, "Ошибка сервера. Попробуйте ещё раз.", http.StatusInternalServerError)
}

type contextKey int

const (
	userKey contextKey = iota
	csrfKey
)

func Token() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}

func TokenHash(s string) string { 
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:]) 
}

func CurrentUser(r *http.Request) *models.User {
	u, _ := r.Context().Value(userKey).(*models.User)
	return u
}

func (a *Manager) Cookie(w http.ResponseWriter, name, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: a.SecureCookies, SameSite: http.SameSiteLaxMode, MaxAge: maxAge})
}

func (a *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; form-action 'self'; frame-ancestors 'none'")
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		r = r.WithContext(ctx)
		csrf := ""
		if c, err := r.Cookie("csrf"); err == nil && len(c.Value) == 64 {
			csrf = c.Value
		}
		if csrf == "" {
			var err error
			csrf, err = Token()
			if err != nil {
				a.fail(w, err)
				return
			}
			a.Cookie(w, "csrf", csrf, 86400)
		}
		if r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Некорректная форма", 400)
				return
			}
			if subtle.ConstantTimeCompare([]byte(r.PostForm.Get("csrf")), []byte(csrf)) != 1 {
				http.Error(w, "Форма устарела. Обновите страницу.", 403)
				return
			}
		}
		ctx = context.WithValue(r.Context(), csrfKey, csrf)
		if c, err := r.Cookie("session"); err == nil {
			u, err := database.GetSessionUser(ctx, a.DB, TokenHash(c.Value))
			if err == nil {
				ctx = context.WithValue(ctx, userKey, &u)
			} else if !errors.Is(err, sql.ErrNoRows) {
				a.fail(w, err)
				return
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Require(teacher bool, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := CurrentUser(r)
		if u == nil {
			http.Redirect(w, r, "/login", 303)
			return
		}
		if teacher && !u.Teacher() {
			http.Error(w, "Доступ только для учителя", 403)
			return
		}
		next(w, r)
	})
}
