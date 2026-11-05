package users

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"lms/backend/pkg/core"
)

type repositoryStub struct {
	user *User
}

func (stub *repositoryStub) Create(_ context.Context, user *User) error {
	stub.user = user
	return nil
}

func (stub *repositoryStub) GetByID(_ context.Context, _ uuid.UUID) (*User, error) {
	return stub.user, nil
}

func (stub *repositoryStub) GetByEmail(_ context.Context, _ string) (*User, error) {
	if stub.user == nil {
		return nil, core.ErrNotFound
	}
	return stub.user, nil
}

func (stub *repositoryStub) Update(_ context.Context, user *User) error {
	stub.user = user
	return nil
}

func (stub *repositoryStub) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (stub *repositoryStub) List(_ context.Context, _ ListParams) ([]User, error) {
	return nil, nil
}

func TestServiceCreateAndAuthenticate(t *testing.T) {
	stub := &repositoryStub{}
	service := NewService(stub)

	user, err := service.Create(context.Background(), CreateInput{
		Name:     "Ada",
		Surname:  "Lovelace",
		Email:    "Ada@example.com",
		Password: "correct horse battery staple",
		Role:     RoleStudent,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.PasswordHash == "correct horse battery staple" || user.PasswordHash == "" {
		t.Fatal("password was not stored as a hash")
	}

	authenticated, err := service.Authenticate(context.Background(), "ada@example.com", "correct horse battery staple")
	if err != nil {
		t.Fatalf("authenticate user: %v", err)
	}
	if authenticated.ID != user.ID {
		t.Fatalf("authenticated user ID = %s, want %s", authenticated.ID, user.ID)
	}
}

func TestServiceRejectsInvalidRole(t *testing.T) {
	service := NewService(&repositoryStub{})

	_, err := service.Create(context.Background(), CreateInput{
		Name:     "Ada",
		Surname:  "Lovelace",
		Email:    "ada@example.com",
		Password: "password",
		Role:     Role("owner"),
	})
	if err != core.ErrValidation {
		t.Fatalf("error = %v, want %v", err, core.ErrValidation)
	}
}
