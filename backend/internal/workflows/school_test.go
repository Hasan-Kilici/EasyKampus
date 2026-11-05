package workflows

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lms/backend/pkg/domain/campuses"
	"lms/backend/pkg/domain/schools"
)

func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&schools.School{}, &schools.Staff{}, &campuses.Campus{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return db
}

func TestSchoolStarterCreatesSchoolAndCampusesAtomically(t *testing.T) {
	db := testDatabase(t)
	starter := NewSchoolStarter(db)

	result, err := starter.Start(context.Background(), StartSchoolInput{
		School: schools.CreateInput{Name: "North Academy", Code: "NORTH"},
		Campuses: []CampusCreateInput{
			{Name: "Main Campus", Code: "MAIN"},
			{Name: "Science Campus", Code: "SCIENCE"},
		},
	})
	if err != nil {
		t.Fatalf("start school: %v", err)
	}
	if result.School == nil || len(result.Campuses) != 2 {
		t.Fatalf("result = %+v, want school and two campuses", result)
	}

	var schoolCount, campusCount int64
	db.Model(&schools.School{}).Count(&schoolCount)
	db.Model(&campuses.Campus{}).Count(&campusCount)
	if schoolCount != 1 || campusCount != 2 {
		t.Fatalf("counts = school %d, campus %d; want 1 and 2", schoolCount, campusCount)
	}
}

func TestSchoolStarterRollsBackOnInvalidCampus(t *testing.T) {
	db := testDatabase(t)
	starter := NewSchoolStarter(db)

	_, err := starter.Start(context.Background(), StartSchoolInput{
		School: schools.CreateInput{Name: "North Academy", Code: "NORTH"},
		Campuses: []CampusCreateInput{
			{Name: "Valid Campus", Code: "VALID"},
			{Name: "", Code: "INVALID"},
		},
	})
	if err == nil {
		t.Fatal("start school succeeded with invalid campus")
	}

	var schoolCount, campusCount int64
	db.Model(&schools.School{}).Count(&schoolCount)
	db.Model(&campuses.Campus{}).Count(&campusCount)
	if schoolCount != 0 || campusCount != 0 {
		t.Fatalf("rollback counts = school %d, campus %d; want 0 and 0", schoolCount, campusCount)
	}
}
