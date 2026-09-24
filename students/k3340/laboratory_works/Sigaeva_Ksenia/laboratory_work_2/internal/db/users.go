package db

import (
	"context"
	"database/sql"
	"homeworkboard/internal/models"
)

func CreateUser(ctx context.Context, conn *sql.DB, username, name, hash, role string) error {
	_, err := conn.ExecContext(ctx, `INSERT INTO users(username,name,password_hash,role) VALUES($1,$2,$3,$4)`, username, name, hash, role)
	return err
}
func GetCredentials(ctx context.Context, conn *sql.DB, username string) (int64, string, error) {
	var id int64
	var hash string
	err := conn.QueryRowContext(ctx, `SELECT id,password_hash FROM users WHERE username=$1`, username).Scan(&id, &hash)
	return id, hash, err
}
func CreateSession(ctx context.Context, conn *sql.DB, hash string, userID int64) error {
	_, err := conn.ExecContext(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '24 hours')`, hash, userID)
	return err
}
func DeleteSession(ctx context.Context, conn *sql.DB, hash string) error {
	_, err := conn.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=$1`, hash)
	return err
}
func GetSessionUser(ctx context.Context, conn *sql.DB, hash string) (models.User, error) {
	var user models.User
	err := conn.QueryRowContext(ctx, `SELECT u.id,u.username,u.name,u.role FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now()`, hash).Scan(&user.ID, &user.Username, &user.Name, &user.Role)
	return user, err
}
