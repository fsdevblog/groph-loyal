package domain

import "time"

type User struct {
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
	Username  string
	Password  string
}
