package handlers

import (
	"fmt"
	"strconv"
	"time"
)

func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("некорректный идентификатор")
	}
	return id, nil
}

func dateValid(issued, due string) bool {
	i, e1 := time.Parse("2006-01-02", issued)
	d, e2 := time.Parse("2006-01-02", due)
	return e1 == nil && e2 == nil && !d.Before(i)
}
