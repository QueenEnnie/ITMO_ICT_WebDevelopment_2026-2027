package db

import (
	"context"
	"database/sql"
	"fmt"
	"homeworkboard/internal/models"
)

func CountAssignments(ctx context.Context, conn *sql.DB, subject int64, query string) (int, error) {
	var count int
	err := conn.QueryRowContext(ctx, `SELECT count(*) FROM assignments a`+assignmentFilter, subject, query).Scan(&count)
	return count, err
}

const assignmentFilter = ` WHERE ($1::bigint=0 OR a.subject_id=$1) AND ($2='' OR a.title ILIKE '%'||$2||'%' OR a.body ILIKE '%'||$2||'%')`

func ListAssignments(ctx context.Context, conn *sql.DB, subject int64, query string, limit, offset int) ([]models.Assignment, error) {
	rows, err := conn.QueryContext(ctx, assignmentSelect+assignmentFilter+` ORDER BY a.due_at,a.id LIMIT $3 OFFSET $4`, subject, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.Assignment
	for rows.Next() {
		item, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func GetSubmissions(ctx context.Context, conn *sql.DB, id int64, user *models.User) ([]models.Submission, error) {
	var result []models.Submission
	query := `SELECT s.id,u.name,s.answer,to_char(s.submitted_at AT TIME ZONE 'Europe/Moscow','DD.MM.YYYY HH24:MI'),s.grade,s.feedback,(s.submitted_at AT TIME ZONE 'Europe/Moscow')::date>a.due_at FROM submissions s JOIN users u ON u.id=s.student_id JOIN assignments a ON a.id=s.assignment_id WHERE s.assignment_id=$1`
	args := []any{id}
	if !user.Teacher() {
		query += ` AND s.student_id=$2`
		args = append(args, user.ID)
	}
	rows, err := conn.QueryContext(ctx, query+` ORDER BY s.submitted_at,s.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s models.Submission
		var g sql.NullInt64
		if err = rows.Scan(&s.ID, &s.Student, &s.Answer, &s.Submitted, &g, &s.Feedback, &s.Late); err != nil {
			return nil, err
		}
		s.Grade = gradeText(g)
		result = append(result, s)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func SaveSubmission(ctx context.Context, conn *sql.DB, id, studentID int64, answer string) (int64, error) {
	result, err := conn.ExecContext(ctx, `INSERT INTO submissions(assignment_id,student_id,answer) VALUES($1,$2,$3) ON CONFLICT(assignment_id,student_id) DO UPDATE SET answer=excluded.answer,submitted_at=now() WHERE submissions.grade IS NULL`, id, studentID, answer)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func DeleteAssignment(ctx context.Context, conn *sql.DB, id int64) (int64, error) {
	result, err := conn.ExecContext(ctx, `DELETE FROM assignments WHERE id=$1`, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func CreateSubject(ctx context.Context, conn *sql.DB, name string) error {
	_, err := conn.ExecContext(ctx, `INSERT INTO subjects(name) VALUES($1)`, name)
	return err
}

func SaveAssignment(ctx context.Context, conn *sql.DB, item models.Assignment, teacherID int64) (int64, error) {
	id := item.ID
	var err error
	if id == 0 {
		err = conn.QueryRowContext(ctx, `INSERT INTO assignments(subject_id,teacher_id,title,body,issued_at,due_at,penalties) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, item.SubjectID, teacherID, item.Title, item.Body, item.Issued, item.Due, item.Penalties).Scan(&id)
	} else {
		_, err = conn.ExecContext(ctx, `UPDATE assignments SET subject_id=$1,title=$2,body=$3,issued_at=$4,due_at=$5,penalties=$6 WHERE id=$7`, item.SubjectID, item.Title, item.Body, item.Issued, item.Due, item.Penalties, id)
	}
	return id, err
}

func GradeSubmission(ctx context.Context, conn *sql.DB, id int64, grade int, feedback string) (int64, error) {
	var assignmentID int64
	err := conn.QueryRowContext(ctx, `UPDATE submissions SET grade=$1,feedback=$2 WHERE id=$3 RETURNING assignment_id`, grade, feedback, id).Scan(&assignmentID)
	return assignmentID, err
}

func GetGradebook(ctx context.Context, conn *sql.DB) ([]models.Assignment, []models.GradeRow, error) {
	var assignments []models.Assignment
	var result []models.GradeRow
	rows, err := conn.QueryContext(ctx, assignmentSelect+` ORDER BY a.id`)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		item, e := scanAssignment(rows)
		if e != nil {
			rows.Close()
			return nil, nil, e
		}
		assignments = append(assignments, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	rows, err = conn.QueryContext(ctx, `SELECT id,name FROM users WHERE role='student' ORDER BY name,id`)
	if err != nil {
		return nil, nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		var name string
		if err = rows.Scan(&id, &name); err != nil {
			rows.Close()
			return nil, nil, err
		}
		ids = append(ids, id)
		result = append(result, models.GradeRow{Name: name})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	grades := make(map[[2]int64]string)
	rows, err = conn.QueryContext(ctx, `SELECT student_id,assignment_id,grade FROM submissions`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var student, assignment int64
		var g sql.NullInt64
		if err = rows.Scan(&student, &assignment, &g); err != nil {
			return nil, nil, err
		}
		grades[[2]int64{student, assignment}] = gradeText(g)
	}
	if err = rows.Err(); err != nil {
		return nil, nil, err
	}
	for i, id := range ids {
		for _, item := range assignments {
			value, ok := grades[[2]int64{id, item.ID}]
			if !ok {
				value = "—"
			}
			result[i].Cells = append(result[i].Cells, value)
		}
	}
	return assignments, result, nil
}

func gradeText(g sql.NullInt64) string {
	if !g.Valid {
		return "На проверке"
	}
	return fmt.Sprint(g.Int64)
}
