package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"homeworkboard/internal/auth"
	database "homeworkboard/internal/db"
	"homeworkboard/internal/models"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const pageSize = 6

func (a *App) list(w http.ResponseWriter, r *http.Request) {
	p := Page{Title: "Задания", Query: strings.TrimSpace(r.URL.Query().Get("q")), SubjectFilter: r.URL.Query().Get("subject"), PageNum: 1}
	var subject int64
	if p.SubjectFilter != "" {
		var err error
		subject, err = strconv.ParseInt(p.SubjectFilter, 10, 64)
		if err != nil || subject < 1 {
			http.Error(w, "Некорректный предмет", 400)
			return
		}
	}
	if s := r.URL.Query().Get("page"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			http.Error(w, "Некорректная страница", 400)
			return
		}
		p.PageNum = n
	}
	var err error
	p.Total, err = database.CountAssignments(r.Context(), a.db, subject, p.Query)
	if err != nil {
		a.fail(w, err)
		return
	}
	p.Pages = (p.Total + pageSize - 1) / pageSize
	if p.Pages == 0 {
		p.Pages = 1
	}
	if p.PageNum > p.Pages {
		p.PageNum = p.Pages
	}
	p.Assignments, err = database.ListAssignments(r.Context(), a.db, subject, p.Query, pageSize, (p.PageNum-1)*pageSize)
	if err != nil {
		a.fail(w, err)
		return
	}
	p.Subjects, err = database.GetSubjects(r.Context(), a.db)
	if err != nil {
		a.fail(w, err)
		return
	}
	pageURL := func(n int) string {
		v := url.Values{"q": {p.Query}, "subject": {p.SubjectFilter}, "page": {strconv.Itoa(n)}}
		return "/assignments?" + v.Encode()
	}
	if p.PageNum > 1 {
		p.Prev = pageURL(p.PageNum - 1)
	}
	if p.PageNum < p.Pages {
		p.Next = pageURL(p.PageNum + 1)
	}
	a.render(w, r, "list", p, 200)
}

func (a *App) detail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	item, err := database.GetAssignment(r.Context(), a.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		a.fail(w, err)
		return
	}
	p := Page{Title: item.Title, Assignment: item}
	p.Submissions, err = database.GetSubmissions(r.Context(), a.db, id, auth.CurrentUser(r))
	if err != nil {
		a.fail(w, err)
		return
	}
	if len(p.Submissions) > 0 {
		p.Submission = p.Submissions[len(p.Submissions)-1]
	}
	a.render(w, r, "detail", p, 200)
}

func (a *App) submit(w http.ResponseWriter, r *http.Request) {
	u := auth.CurrentUser(r)
	if u.Teacher() {
		http.Error(w, "Ответы отправляют ученики", 403)
		return
	}
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_, err = database.GetAssignment(r.Context(), a.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		a.fail(w, err)
		return
	}
	answer := strings.TrimSpace(r.PostForm.Get("answer"))
	if answer == "" || utf8.RuneCountInString(answer) > 10000 {
		http.Error(w, "Ответ должен содержать от 1 до 10000 символов. Вернитесь к форме.", 422)
		return
	}
	n, err := database.SaveSubmission(r.Context(), a.db, id, u.ID, answer)
	if err != nil {
		a.fail(w, err)
		return
	}
	if n == 0 {
		http.Error(w, "Ответ уже оценён и недоступен для редактирования", 409)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/assignments/%d", id), 303)
}

func (a *App) edit(w http.ResponseWriter, r *http.Request) {
	p := Page{Title: "Новое задание", Form: map[string]string{"issued": time.Now().Format("2006-01-02"), "due": time.Now().AddDate(0, 0, 7).Format("2006-01-02")}}
	var id int64
	var err error
	if raw := r.PathValue("id"); raw != "" {
		id, err = parseID(raw)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		p.Assignment, err = database.GetAssignment(r.Context(), a.db, id)
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			a.fail(w, err)
			return
		}
		p.Title = "Редактирование задания"
		x := p.Assignment
		p.Form = map[string]string{"title": x.Title, "body": x.Body, "subject": fmt.Sprint(x.SubjectID), "issued": x.Issued, "due": x.Due, "penalties": x.Penalties}
	}
	p.Subjects, err = database.GetSubjects(r.Context(), a.db)
	if err != nil {
		a.fail(w, err)
		return
	}
	if r.Method == "GET" {
		a.render(w, r, "edit", p, 200)
		return
	}
	for _, key := range []string{"title", "body", "subject", "issued", "due", "penalties"} {
		p.Form[key] = strings.TrimSpace(r.PostForm.Get(key))
	}
	f := p.Form
	if !dateValid(f["issued"], f["due"]) {
		p.Error = "Укажите корректные даты: срок сдачи не может быть раньше даты выдачи."
		a.render(w, r, "edit", p, 422)
		return
	}
	subjectID, e := strconv.ParseInt(f["subject"], 10, 64)
	subjectOK := false
	for _, s := range p.Subjects {
		if s.ID == subjectID {
			subjectOK = true
		}
	}
	if e != nil || !subjectOK || f["title"] == "" || utf8.RuneCountInString(f["title"]) > 150 || f["body"] == "" || utf8.RuneCountInString(f["body"]) > 10000 || utf8.RuneCountInString(f["penalties"]) > 2000 {
		p.Error = "Проверьте предмет, название (до 150 символов), текст (до 10000), штрафы (до 2000)."
		a.render(w, r, "edit", p, 422)
		return
	}
	item := models.Assignment{ID: id, SubjectID: subjectID, Title: f["title"], Body: f["body"], Issued: f["issued"], Due: f["due"], Penalties: f["penalties"]}
	id, err = database.SaveAssignment(r.Context(), a.db, item, auth.CurrentUser(r).ID)
	if err != nil {
		a.fail(w, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/assignments/%d", id), 303)
}

func (a *App) deleteAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if r.PostForm.Get("confirm") != "yes" {
		http.Error(w, "Подтвердите удаление", 422)
		return
	}
	n, err := database.DeleteAssignment(r.Context(), a.db, id)
	if err != nil {
		a.fail(w, err)
		return
	}
	if n == 0 {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/assignments", 303)
}

func (a *App) grade(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	g, err := strconv.Atoi(r.PostForm.Get("grade"))
	feedback := strings.TrimSpace(r.PostForm.Get("feedback"))
	if err != nil || g < 2 || g > 5 || utf8.RuneCountInString(feedback) > 2000 {
		http.Error(w, "Оценка от 2 до 5; комментарий до 2000 символов", 422)
		return
	}
	assignmentID, err := database.GradeSubmission(r.Context(), a.db, id, g, feedback)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		a.fail(w, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/assignments/%d", assignmentID), 303)
}

func (a *App) subjects(w http.ResponseWriter, r *http.Request) {
	p := Page{Title: "Предметы"}
	status := 200
	if r.Method == "POST" {
		name := strings.TrimSpace(r.PostForm.Get("name"))
		if name == "" || utf8.RuneCountInString(name) > 100 {
			p.Error = "Введите название до 100 символов."
			status = 422
		} else {
			err := database.CreateSubject(r.Context(), a.db, name)
			if database.Duplicate(err) {
				p.Error = "Предмет уже существует."
				status = 422
			} else if err != nil {
				a.fail(w, err)
				return
			} else {
				http.Redirect(w, r, "/teacher/subjects", 303)
				return
			}
		}
	}
	var err error
	p.Subjects, err = database.GetSubjects(r.Context(), a.db)
	if err != nil {
		a.fail(w, err)
		return
	}
	a.render(w, r, "subjects", p, status)
}

func (a *App) gradebook(w http.ResponseWriter, r *http.Request) {
	p := Page{Title: "Журнал оценок"}
	var err error
	p.Assignments, p.Rows, err = database.GetGradebook(r.Context(), a.db)
	if err != nil {
		a.fail(w, err)
		return
	}
	a.render(w, r, "gradebook", p, 200)
}
