package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	UUID         uuid.UUID `json:"uuid" db:"uuid"`
	Login        string    `json:"login" db:"login"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	Balance      float64   `json:"balance" db:"balance"`
	Withdrawn    float64   `json:"withdrawn" db:"withdrawn"`
}

type Order struct {
	OrderNumber string    `json:"number" db:"order_num"`
	Status      string    `json:"status" db:"status"`
	Accrual     *float64  `json:"accrual" db:"accrual"`
	Withdrawn   *float64  `json:"withdrawn" db:"withdrawn"`
	UserUUID    uuid.UUID `json:"user_uuid" db:"user_uuid"`
	CreatedAt   time.Time `json:"uploaded_at" db:"created_at"`
}
