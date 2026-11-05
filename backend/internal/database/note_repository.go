package database

import (
	"context"

	"github.com/google/uuid"

	"gorm.io/gorm"

	"lms/backend/pkg/domain/notes"
)

type NoteRepository struct {
	db *gorm.DB
}

func NewNoteRepository(db *gorm.DB) *NoteRepository {
	return &NoteRepository{db: db}
}

func (repository *NoteRepository) Create(ctx context.Context, note *notes.Note) error {
	return translate(repository.db.WithContext(ctx).Create(note).Error)
}

func (repository *NoteRepository) GetByID(ctx context.Context, id uuid.UUID) (*notes.Note, error) {
	var note notes.Note
	err := repository.db.WithContext(ctx).First(&note, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &note, nil
}

func (repository *NoteRepository) Update(ctx context.Context, note *notes.Note) error {
	return translate(repository.db.WithContext(ctx).Save(note).Error)
}

func (repository *NoteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&notes.Note{}, "id = ?", id).Error)
}

func (repository *NoteRepository) List(ctx context.Context, params notes.ListParams) ([]notes.Note, error) {
	if params.Limit < 1 || params.Limit > 200 {
		params.Limit = 50
	}
	query := repository.db.WithContext(ctx).Model(&notes.Note{})
	if params.ScopeApplied && len(params.ClassIDs) == 0 {
		return []notes.Note{}, nil
	}
	if params.AuthorID != nil {
		query = query.Where("author_id = ?", *params.AuthorID)
	}
	if params.StudentID != nil {
		query = query.Where("student_id = ?", *params.StudentID)
	}
	if params.ClassID != nil {
		query = query.Where("class_id = ?", *params.ClassID)
	}
	if len(params.ClassIDs) > 0 {
		query = query.Where("class_id IN ?", params.ClassIDs)
	}
	if params.CourseID != nil {
		query = query.Where("course_id = ?", *params.CourseID)
	}
	if params.SharedOnly {
		query = query.Where("visible_to_student = ?", true)
	}
	var result []notes.Note
	err := query.Order("created_at DESC").Limit(params.Limit).Find(&result).Error
	return result, translate(err)
}

func (repository *NoteRepository) CountByTeacher(ctx context.Context, authorID uuid.UUID) (int64, error) {
	var count int64
	err := repository.db.WithContext(ctx).Model(&notes.Note{}).
		Where("author_id = ?", authorID).
		Count(&count).Error
	return count, translate(err)
}
