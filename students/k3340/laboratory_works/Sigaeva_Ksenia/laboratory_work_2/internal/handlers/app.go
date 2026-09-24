package handlers

import (
	"bytes"
	"database/sql"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	assets "homeworkboard"
	"homeworkboard/internal/auth"
	"homeworkboard/internal/models"
)

type App struct {
	db    *sql.DB
	views map[string]*template.Template
	auth  *auth.Manager
}

func New(conn *sql.DB, secureCookies bool) (*App, error) {
	views, err := loadViews()
	if err != nil {
		return nil, err
	}
	return &App{db: conn, views: views, auth: &auth.Manager{DB: conn, SecureCookies: secureCookies}}, nil
}

func loadViews() (map[string]*template.Template, error) {
	paths := map[string]string{
		"login":     "auth/login.html",
		"register":  "auth/register.html",
		"list":      "assignments/list.html",
		"detail":    "assignments/detail.html",
		"edit":      "assignments/edit.html",
		"gradebook": "gradebook/gradebook.html",
		"subjects":  "subjects/subjects.html",
	}
	views := make(map[string]*template.Template)
	for name, path := range paths {
		view, err := template.ParseFS(assets.Files, "templates/base.html", "templates/"+path)
		if err != nil {
			return nil, err
		}
		views[name] = view
	}
	return views, nil
}

type Page struct {
	Title, Error, CSRF, Query, SubjectFilter, Prev, Next string
	User                                                 *models.User
	Assignments                                          []models.Assignment
	Subjects                                             []models.Subject
	Assignment                                           models.Assignment
	Submission                                           models.Submission
	Submissions                                          []models.Submission
	Rows                                                 []models.GradeRow
	Form                                                 map[string]string
	PageNum, Pages, Total                                int
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(assets.Files, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/assignments", http.StatusSeeOther) })
	mux.HandleFunc("GET /login", a.login)
	mux.HandleFunc("POST /login", a.login)
	mux.HandleFunc("GET /register", a.register)
	mux.HandleFunc("POST /register", a.register)
	mux.HandleFunc("POST /logout", a.logout)
	mux.Handle("GET /assignments", auth.Require(false, a.list))
	mux.Handle("GET /assignments/{id}", auth.Require(false, a.detail))
	mux.Handle("POST /assignments/{id}/submit", auth.Require(false, a.submit))
	mux.Handle("GET /gradebook", auth.Require(false, a.gradebook))
	mux.Handle("GET /teacher/assignments/new", auth.Require(true, a.edit))
	mux.Handle("POST /teacher/assignments/new", auth.Require(true, a.edit))
	mux.Handle("GET /teacher/assignments/{id}/edit", auth.Require(true, a.edit))
	mux.Handle("POST /teacher/assignments/{id}/edit", auth.Require(true, a.edit))
	mux.Handle("POST /teacher/assignments/{id}/delete", auth.Require(true, a.deleteAssignment))
	mux.Handle("POST /teacher/submissions/{id}/grade", auth.Require(true, a.grade))
	mux.Handle("GET /teacher/subjects", auth.Require(true, a.subjects))
	mux.Handle("POST /teacher/subjects", auth.Require(true, a.subjects))
	return http.NewCrossOriginProtection().Handler(a.auth.Middleware(mux))
}

func (a *App) render(w http.ResponseWriter, r *http.Request, name string, p Page, status int) {
	p.User = auth.CurrentUser(r)
	p.CSRF = auth.CSRF(r)
	var out bytes.Buffer
	if err := a.views[name].ExecuteTemplate(&out, "base", p); err != nil {
		a.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(out.Bytes())
}
func (a *App) fail(w http.ResponseWriter, err error) {
	log.Print(err)
	http.Error(w, "Ошибка сервера. Попробуйте ещё раз.", http.StatusInternalServerError)
}
