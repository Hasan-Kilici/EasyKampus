package workflows

import (
	"context"

	"gorm.io/gorm"

	"lms/backend/internal/database"
	"lms/backend/pkg/domain/campuses"
	"lms/backend/pkg/domain/schools"
)

type StartSchoolInput struct {
	School   schools.CreateInput `json:"school"`
	Campuses []CampusCreateInput `json:"campuses"`
}

type CampusCreateInput struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type StartSchoolResult struct {
	School   *schools.School   `json:"school"`
	Campuses []campuses.Campus `json:"campuses"`
}

type SchoolStarter interface {
	Start(ctx context.Context, input StartSchoolInput) (*StartSchoolResult, error)
}

type schoolStarter struct{ db *gorm.DB }

func NewSchoolStarter(db *gorm.DB) SchoolStarter { return &schoolStarter{db: db} }

func (starter *schoolStarter) Start(ctx context.Context, input StartSchoolInput) (*StartSchoolResult, error) {
	result := &StartSchoolResult{}
	err := starter.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		schoolRepository := database.NewSchoolRepository(tx)
		campusRepository := database.NewCampusRepository(tx)

		schoolService := schools.NewService(schoolRepository)
		school, err := schoolService.Create(ctx, input.School)
		if err != nil {
			return err
		}
		result.School = school

		for _, campusInput := range input.Campuses {
			campusService := campuses.NewService(campusRepository, schoolRepository)
			campus, err := campusService.Create(ctx, campuses.CreateInput{
				SchoolID: school.ID.String(),
				Name:     campusInput.Name,
				Code:     campusInput.Code,
			})
			if err != nil {
				return err
			}
			result.Campuses = append(result.Campuses, *campus)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
