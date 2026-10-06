package model

import (
	"time"

	"github.com/google/uuid"
)

const PersonColumns = "id, email, phone, first_name, last_name, created_at, updated_at, version"

type Person struct {
	ID        uuid.UUID `db:"id"`
	Email     string    `db:"email"`
	Phone     string    `db:"phone"`
	FirstName string    `db:"first_name"`
	LastName  string    `db:"last_name"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	Version   int64     `db:"version"`
}
