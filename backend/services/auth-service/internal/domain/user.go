package domain

import "errors"

var ErrCredentials = errors.New("invalid credentials")
var ErrConflict = errors.New("account unavailable")
var ErrValidation = errors.New("invalid input")

type User struct {
	ID       string `db:"id"`
	Email    string `db:"email"`
	Role     string `db:"role"`
	Password string `db:"password_hash"`
}
type Session struct {
	Access, Refresh string
	User            User
}
