package database

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/pkg/domain/courses"
	"lms/backend/pkg/domain/topics"
)

type TopicRepository struct {
	db *gorm.DB
}

func NewTopicRepository(db *gorm.DB) *TopicRepository {
	return &TopicRepository{db: db}
}

func (repository *TopicRepository) Create(ctx context.Context, topic *topics.Topic) error {
	return translate(repository.db.WithContext(ctx).Create(topic).Error)
}

func (repository *TopicRepository) GetByID(ctx context.Context, id uuid.UUID) (*topics.Topic, error) {
	var topic topics.Topic
	err := repository.db.WithContext(ctx).First(&topic, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}
	return &topic, nil
}

func (repository *TopicRepository) Update(ctx context.Context, topic *topics.Topic) error {
	return translate(repository.db.WithContext(ctx).Save(topic).Error)
}

func (repository *TopicRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return translate(repository.db.WithContext(ctx).Delete(&topics.Topic{}, "id = ?", id).Error)
}

func (repository *TopicRepository) ListByCourse(ctx context.Context, courseID uuid.UUID, activeOnly bool) ([]topics.Topic, error) {
	query := repository.db.WithContext(ctx).Where("course_id = ?", courseID)
	if activeOnly {
		query = query.Where("active = ?", true)
	}
	var result []topics.Topic
	err := query.Order("weight DESC, name ASC").Find(&result).Error
	return result, translate(err)
}

func (repository *TopicRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return exists(ctx, repository.db, &topics.Topic{}, "id = ?", id)
}

func (repository *TopicRepository) CourseExists(ctx context.Context, courseID uuid.UUID) (bool, error) {
	return exists(ctx, repository.db, &courses.Course{}, "id = ?", courseID)
}

func (repository *TopicRepository) CourseIDForTopic(ctx context.Context, topicID uuid.UUID) (uuid.UUID, error) {
	topic, err := repository.GetByID(ctx, topicID)
	if err != nil {
		return uuid.Nil, err
	}
	return topic.CourseID, nil
}
