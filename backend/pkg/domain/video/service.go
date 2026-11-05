package video

import (
	"context"

	"github.com/google/uuid"
)

type Service interface {
	CreateVideo(ctx context.Context, video *Video) error
	GetVideoByID(ctx context.Context, id uuid.UUID) (*Video, error)
	UpdateVideo(ctx context.Context, video *Video) error
	DeleteVideo(ctx context.Context, id uuid.UUID) error
	ListVideosByLesson(ctx context.Context, params ListParams) ([]Video, error)
	UpsertVideoCompletions(ctx context.Context, completions []VideoCompletion) error
	ListVideoCompletions(ctx context.Context, studentID uuid.UUID, lessonID *uuid.UUID) ([]VideoCompletion, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateVideo(ctx context.Context, video *Video) error {
	return s.repo.Create(ctx, video)
}

func (s *service) GetVideoByID(ctx context.Context, id uuid.UUID) (*Video, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) UpdateVideo(ctx context.Context, video *Video) error {
	return s.repo.Update(ctx, video)
}

func (s *service) DeleteVideo(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) ListVideosByLesson(ctx context.Context, params ListParams) ([]Video, error) {
	return s.repo.ListByLesson(ctx, params)
}

func (s *service) UpsertVideoCompletions(ctx context.Context, completions []VideoCompletion) error {
	return s.repo.UpsertCompletions(ctx, completions)
}

func (s *service) ListVideoCompletions(ctx context.Context, studentID uuid.UUID, lessonID *uuid.UUID) ([]VideoCompletion, error) {
	return s.repo.ListCompletions(ctx, studentID, lessonID)
}
