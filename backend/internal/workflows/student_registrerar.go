package workflows

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"lms/backend/internal/database"
	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/campuses"
	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/enrollments"
	"lms/backend/pkg/domain/schools"
	"lms/backend/pkg/domain/terms"
	"lms/backend/pkg/domain/users"
)

// Otomatik kayıt yapılamadığında nedeni (hesap yine de oluşturulur).
const (
	SkipNoSchool     = "no_school"      // sistemde okul yok
	SkipNoCampus     = "no_campus"      // okul var ama hiçbirinde kampüs yok
	SkipNoActiveTerm = "no_active_term" // kampüs var ama okulun aktif dönemi yok
	SkipNoClass      = "no_class"       // kampüste sınıf yok ve otomatik sınıf kapalı
)

// Varsayılan sınıf, kampüste hiç sınıf yoksa otomatik açılır (autoCreateClass).
const (
	defaultClassName  = "Genel"
	defaultClassLevel = "Genel"
)

type RegisterStudentInput struct {
	Name     string
	Surname  string
	Email    string
	Password string
	Age      int
	// İsteğe bağlı: boşsa en eski okul / kampüs otomatik seçilir.
	SchoolID string
	CampusID string
}

type Placement struct {
	SchoolID uuid.UUID `json:"school_id"`
	CampusID uuid.UUID `json:"campus_id"`
	ClassID  uuid.UUID `json:"class_id"`
	Term     string    `json:"term"`
}

type RegisterStudentResult struct {
	User          *users.User
	Enrollment    *enrollments.Enrollment
	Placement     *Placement
	SkippedReason string // boşsa öğrenci sınıfa kaydedildi
}

type StudentRegistrar interface {
	Register(ctx context.Context, input RegisterStudentInput) (*RegisterStudentResult, error)
}

type studentRegistrar struct {
	db              *gorm.DB
	autoCreateClass bool
}

// NewStudentRegistrar öğrenci hesabını ve (mümkünse) ilk sınıf kaydını tek transaction'da oluşturur.
func NewStudentRegistrar(db *gorm.DB) StudentRegistrar {
	return &studentRegistrar{db: db, autoCreateClass: true}
}

func (registrar *studentRegistrar) Register(ctx context.Context, input RegisterStudentInput) (*RegisterStudentResult, error) {
	result := &RegisterStudentResult{}
	err := registrar.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := users.NewService(database.NewUserRepository(tx)).Create(ctx, users.CreateInput{
			Name: input.Name, Surname: input.Surname, Email: input.Email,
			Password: input.Password, Age: input.Age, Role: users.RoleStudent,
		})
		if err != nil {
			return err
		}
		result.User = user

		placement, reason, err := registrar.choosePlacement(ctx, tx, input)
		if err != nil {
			return err
		}
		if placement == nil {
			result.SkippedReason = reason
			return nil // hesap oluşur, kayıt yönetici tarafından sonradan yapılır
		}

		enrollment := &enrollments.Enrollment{
			StudentID:  user.ID,
			ClassID:    placement.ClassID,
			Term:       placement.Term,
			Status:     enrollments.StatusActive,
			EnrolledAt: time.Now().UTC(),
			Note:       "Otomatik kayıt",
		}
		if err := enrollment.Validate(); err != nil {
			return err
		}
		if err := database.NewEnrollmentRepository(tx).Create(ctx, enrollment); err != nil {
			return err
		}
		result.Enrollment = enrollment
		result.Placement = placement
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// choosePlacement: okul + kampüs + aktif dönem + sınıf seçer. Seçilemezse (nil, neden, nil) döner.
func (registrar *studentRegistrar) choosePlacement(ctx context.Context, tx *gorm.DB, input RegisterStudentInput) (*Placement, string, error) {
	wantSchool, err := optionalUUID(input.SchoolID)
	if err != nil {
		return nil, "", core.ErrValidation
	}
	wantCampus, err := optionalUUID(input.CampusID)
	if err != nil {
		return nil, "", core.ErrValidation
	}

	schoolRepository := database.NewSchoolRepository(tx)
	campusRepository := database.NewCampusRepository(tx)
	termRepository := database.NewTermRepository(tx)

	schoolList, err := schoolRepository.List(ctx, schools.ListParams{Limit: 200})
	if err != nil {
		return nil, "", err
	}
	if wantSchool != nil {
		schoolList = slices.DeleteFunc(schoolList, func(school schools.School) bool { return school.ID != *wantSchool })
		if len(schoolList) == 0 {
			return nil, "", core.ErrNotFound
		}
	}
	if len(schoolList) == 0 {
		return nil, SkipNoSchool, nil
	}
	slices.SortFunc(schoolList, func(a, b schools.School) int { return a.CreatedAt.Compare(b.CreatedAt) })

	reason := SkipNoCampus
	for _, school := range schoolList {
		schoolID := school.ID
		campusList, err := campusRepository.List(ctx, campuses.ListParams{SchoolID: &schoolID, Limit: 200})
		if err != nil {
			return nil, "", err
		}
		if wantCampus != nil {
			campusList = slices.DeleteFunc(campusList, func(campus campuses.Campus) bool { return campus.ID != *wantCampus })
			if len(campusList) == 0 && wantSchool != nil {
				return nil, "", core.ErrValidation // istenen kampüs bu okula ait değil
			}
		}
		if len(campusList) == 0 {
			continue
		}
		slices.SortFunc(campusList, func(a, b campuses.Campus) int { return a.CreatedAt.Compare(b.CreatedAt) })

		termName, err := activeTermName(ctx, termRepository, schoolID)
		if err != nil {
			return nil, "", err
		}
		if termName == "" {
			reason = SkipNoActiveTerm
			continue
		}

		campus := campusList[0]
		class, err := registrar.pickClass(ctx, tx, schoolID, campus.ID)
		if err != nil {
			return nil, "", err
		}
		if class == nil {
			reason = SkipNoClass
			continue
		}
		return &Placement{SchoolID: schoolID, CampusID: campus.ID, ClassID: class.ID, Term: termName}, "", nil
	}
	return nil, reason, nil
}

// pickClass kampüsteki en eski sınıfı verir; yoksa (ve izin varsa) "Genel" sınıfını açar.
func (registrar *studentRegistrar) pickClass(ctx context.Context, tx *gorm.DB, schoolID, campusID uuid.UUID) (*classes.Class, error) {
	schoolRepository := database.NewSchoolRepository(tx)
	campusRepository := database.NewCampusRepository(tx)
	classRepository := database.NewClassRepository(tx, database.NewAccessRepository(tx))

	list, err := classRepository.List(ctx,
		classes.ListParams{SchoolID: &schoolID, CampusID: &campusID, Limit: 200},
		classes.Scope{Role: users.RoleSuperAdmin}, // kapsam filtresi yok: tüm sınıfları gör
	)
	if err != nil {
		return nil, err
	}
	if len(list) > 0 {
		slices.SortFunc(list, func(a, b classes.Class) int { return a.CreatedAt.Compare(b.CreatedAt) })
		return &list[0], nil
	}
	if !registrar.autoCreateClass {
		return nil, nil
	}

	service := classes.NewService(classRepository, schoolRepository, campusRepository)
	// (school_id, level, no) benzersiz: aynı okulda başka kampüste "Genel/A" varsa sıradaki harfi dene.
	for letter := 'A'; letter <= 'Z'; letter++ {
		class, err := service.Create(ctx, classes.CreateInput{
			SchoolID: schoolID.String(), CampusID: campusID.String(),
			Name: defaultClassName, Level: defaultClassLevel, No: string(letter),
		})
		if err == nil {
			return class, nil
		}
		if !errors.Is(err, core.ErrAlreadyExists) {
			return nil, err
		}
	}
	return nil, core.ErrAlreadyExists
}

func activeTermName(ctx context.Context, repository *database.TermRepository, schoolID uuid.UUID) (string, error) {
	list, err := repository.List(ctx, schoolID)
	if err != nil {
		return "", err
	}
	for _, term := range list {
		if term.Status == terms.StatusActive {
			return term.Name, nil
		}
	}
	return "", nil
}

func optionalUUID(value string) (*uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
