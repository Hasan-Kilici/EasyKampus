package users

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

type CreateInput struct {
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Age      int    `json:"age"`
	Role     Role   `json:"role"`
}

type UpdateInput struct {
	Name     string `json:"name"`
	Surname  string `json:"surname"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Age      int    `json:"age"`
	Role     Role   `json:"role"`
}

type Service interface {
	Create(ctx context.Context, input CreateInput) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	Authenticate(ctx context.Context, email string, password string) (*User, error)
	List(ctx context.Context, params ListParams) ([]User, error)
	Update(ctx context.Context, id string, input UpdateInput) (*User, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &service{repository: repository}
}

func (service *service) Create(ctx context.Context, input CreateInput) (*User, error) {
	passwordHash, err := core.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &User{
		Name:         strings.TrimSpace(input.Name),
		Surname:      strings.TrimSpace(input.Surname),
		Email:        strings.ToLower(strings.TrimSpace(input.Email)),
		EmailHash:    core.HashEmail(input.Email),
		PasswordHash: passwordHash,
		Age:          input.Age,
		Role:         input.Role,
	}
	if err := user.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (service *service) GetByID(ctx context.Context, id string) (*User, error) {
	parsedID, err := parseID(id)
	if err != nil {
		return nil, core.ErrValidation
	}
	return service.repository.GetByID(ctx, parsedID)
}

func (service *service) Authenticate(ctx context.Context, email string, password string) (*User, error) {
	user, err := service.repository.GetByEmail(ctx, core.HashEmail(email))
	if err != nil {
		if err == core.ErrNotFound {
			return nil, core.ErrUnauthorized
		}
		return nil, err
	}
	if core.CheckPassword(user.PasswordHash, password) != nil {
		return nil, core.ErrUnauthorized
	}
	return user, nil
}

func (service *service) Update(ctx context.Context, id string, input UpdateInput) (*User, error) {
	parsedID, err := parseID(id)
	if err != nil {
		fmt.Println("Error parsing user ID:", err) // Debugging line
		return nil, core.ErrValidation
	}
	user, err := service.repository.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}

	user.Name = strings.TrimSpace(input.Name)
	user.Surname = strings.TrimSpace(input.Surname)
	user.Email = strings.ToLower(strings.TrimSpace(input.Email))
	user.EmailHash = core.HashEmail(input.Email)
	user.Age = input.Age
	user.Role = input.Role
	if input.Password != "" {
		passwordHash, hashErr := core.HashPassword(input.Password)
		if hashErr != nil {
			return nil, hashErr
		}
		user.PasswordHash = passwordHash
	}
	if err := user.Validate(); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (service *service) List(ctx context.Context, params ListParams) ([]User, error) {
	page := params.Page
	if page < 1 {
		page = 1
	}
	limit := params.Limit
	if limit < 1 || limit > 200 {
		limit = 20
	}
	return service.repository.List(ctx, ListParams{Page: page, Limit: limit})
}

func (service *service) Delete(ctx context.Context, id string) error {
	parsedID, err := parseID(id)
	if err != nil {
		return core.ErrValidation
	}
	return service.repository.Delete(ctx, parsedID)
}

func parseID(value string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(value))
}
