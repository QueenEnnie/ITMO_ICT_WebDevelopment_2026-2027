package db

import (
	"context"
	"database/sql"
	"homeworkboard/internal/models"
)

const assignmentSelect = `SELECT a.id,a.subject_id,a.title,a.body,s.name,u.name,
 to_char(a.issued_at,'YYYY-MM-DD'),to_char(a.due_at,'YYYY-MM-DD'),a.penalties,a.due_at<(now() AT TIME ZONE 'Europe/Moscow')::date
 FROM assignments a JOIN subjects s ON s.id=a.subject_id JOIN users u ON u.id=a.teacher_id`

type scanner interface{ Scan(...any) error }

func scanAssignment(row scanner) (models.Assignment, error) {
	var a models.Assignment
	err := row.Scan(&a.ID, &a.SubjectID, &a.Title, &a.Body, &a.Subject, &a.Teacher, &a.Issued, &a.Due, &a.Penalties, &a.Late)
	return a, err
}
func GetAssignment(ctx context.Context, conn *sql.DB, id int64) (models.Assignment, error) {
	return scanAssignment(conn.QueryRowContext(ctx, assignmentSelect+" WHERE a.id=$1", id))
}
func GetSubjects(ctx context.Context, conn *sql.DB) ([]models.Subject, error) {
	rows, err := conn.QueryContext(ctx, `SELECT id,name FROM subjects ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []models.Subject
	for rows.Next() {
		var s models.Subject
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}
