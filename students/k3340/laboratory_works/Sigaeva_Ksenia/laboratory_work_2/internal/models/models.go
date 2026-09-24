package models

import "regexp"

var usernamePattern = regexp.MustCompile(`^[a-z0-9_]{3,40}$`)

func ValidUsername(name string) bool { return usernamePattern.MatchString(name) }

type User struct {
	ID                   int64
	Username, Name, Role string
}

func (u *User) Teacher() bool { return u != nil && u.Role == "teacher" }

type Subject struct {
	ID   int64
	Name string
}
type Assignment struct {
	ID, SubjectID                                         int64
	Title, Body, Subject, Teacher, Issued, Due, Penalties string
	Late                                                  bool
}
type Submission struct {
	ID                                          int64
	Student, Answer, Submitted, Grade, Feedback string
	Late                                        bool
}
type GradeRow struct {
	Name  string
	Cells []string
}
