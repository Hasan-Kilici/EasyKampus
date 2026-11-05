package users

import (
	"context"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type Role string

const (
	RoleSuperAdmin Role = "superadmin"
	RoleAdmin      Role = "admin"
	RoleTeacher    Role = "teacher"
	RoleStudent    Role = "student"
)

type User struct {
	base.Model
	Name         string `gorm:"size:100;not null" json:"name"`
	Surname      string `gorm:"size:100;not null" json:"surname"`
	Email        string `gorm:"-" json:"-"`
	EmailHash    string `gorm:"size:64;not null;uniqueIndex" json:"-"`
	PasswordHash string `gorm:"size:255;not null" json:"-"`
	Age          int    `json:"age"`
	Role         Role   `gorm:"size:20;not null;index" json:"role"`
}

type ListParams struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, params ListParams) ([]User, error)
}
