package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/sessions"
	"lms/backend/pkg/domain/users"
)

const (
	UserIDLocal    = "auth.user_id"
	SessionIDLocal = "auth.session_id"
	RoleLocal      = "auth.role"
)

type ContentScope interface {
	QuizScope(ctx context.Context, quizID uuid.UUID) (uuid.UUID, uuid.UUID, error)
	QuestionQuiz(ctx context.Context, questionID uuid.UUID) (uuid.UUID, error)
	OptionQuestion(ctx context.Context, optionID uuid.UUID) (uuid.UUID, error)
	CourseIDForLesson(ctx context.Context, lessonID uuid.UUID) (uuid.UUID, error)
	TopicCourse(ctx context.Context, topicID uuid.UUID) (uuid.UUID, error)
}

type ResourceScope interface {
	HasAccess(ctx context.Context, userID uuid.UUID, classID uuid.UUID) (bool, error)
	HasTeacherAccess(ctx context.Context, userID uuid.UUID, classID uuid.UUID) (bool, error)
}

type ClassResolver interface {
	ClassIDForResource(ctx context.Context, resource string, id uuid.UUID) (uuid.UUID, error)
	OwnerIDForResource(ctx context.Context, resource string, id uuid.UUID) (uuid.UUID, error)
}

func Auth(secret string, sessionService sessions.Service, stateless bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get(fiber.HeaderAuthorization)
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return fiber.ErrUnauthorized
		}

		claims, err := core.ParseToken(secret, parts[1])
		if err != nil {
			return fiber.ErrUnauthorized
		}
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			return fiber.ErrUnauthorized
		}
		if !stateless {
			session, err := sessionService.GetActive(c.UserContext(), claims.SessionID)
			if err != nil || session.UserID != userID {
				return fiber.ErrUnauthorized
			}
			c.Locals(SessionIDLocal, session.ID)
		} else {
			sessionID, err := uuid.Parse(claims.SessionID)
			if err != nil {
				return fiber.ErrUnauthorized
			}
			c.Locals(SessionIDLocal, sessionID)
		}
		role := users.Role(claims.Role)
		if !role.IsValid() {
			return fiber.ErrUnauthorized
		}
		c.Locals(UserIDLocal, userID)
		c.Locals(RoleLocal, role)
		return c.Next()
	}
}

func RequireRoles(allowed ...users.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals(RoleLocal).(users.Role)
		if !ok {
			return fiber.ErrUnauthorized
		}
		for _, candidate := range allowed {
			if role == candidate {
				return c.Next()
			}
		}
		return fiber.ErrForbidden
	}
}

func SchoolAccess(repository classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			if role == users.RoleSuperAdmin || role == users.RoleAdmin {
				return nil
			}
			schoolID, err := paramID(c, "schoolID")
			if err != nil {
				return err
			}
			allowed, err := repository.HasSchoolAccess(c.UserContext(), userID, schoolID)
			return allowIf(allowed, err)
		})
	}
}

func SchoolWriteAccess(repository classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			schoolID, err := paramID(c, "schoolID")
			if err != nil {
				return err
			}
			if role == users.RoleSuperAdmin {
				return nil
			}
			if role != users.RoleAdmin {
				return fiber.ErrForbidden
			}
			allowed, err := repository.HasSchoolStaffAccess(c.UserContext(), userID, schoolID)
			return allowIf(allowed, err)
		})
	}
}

func CampusAccess(repository classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			if role == users.RoleSuperAdmin || role == users.RoleAdmin {
				return nil
			}
			campusID, err := paramID(c, "campusID")
			if err != nil {
				return err
			}
			allowed, err := repository.HasCampusAccess(c.UserContext(), userID, campusID)
			return allowIf(allowed, err)
		})
	}
}

func CampusWriteAccess(repository classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			if role == users.RoleSuperAdmin {
				return nil
			}
			if role != users.RoleAdmin {
				return fiber.ErrForbidden
			}
			campusID, err := paramID(c, "campusID")
			if err != nil {
				return err
			}
			allowed, err := repository.HasCampusAccess(c.UserContext(), userID, campusID)
			return allowIf(allowed, err)
		})
	}
}

func ClassAccess(repository classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			if role == users.RoleSuperAdmin || role == users.RoleAdmin {
				return nil
			}
			classID, err := paramID(c, "classID")
			if err != nil {
				return err
			}
			allowed, err := repository.HasAccess(c.UserContext(), userID, classID)
			return allowIf(allowed, err)
		})
	}
}

func ClassWriteAccess(repository classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			classID, err := paramID(c, "classID")
			if err != nil {
				return err
			}
			if role == users.RoleSuperAdmin {
				return nil
			}
			if role != users.RoleTeacher {
				return fiber.ErrForbidden
			}
			allowed, err := repository.HasTeacherAccess(c.UserContext(), userID, classID)
			return allowIf(allowed, err)
		})
	}
}

func ClassManagementAccess(repository classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			classID, err := paramID(c, "classID")
			if err != nil {
				return err
			}

			if role == users.RoleSuperAdmin {
				return nil
			}
			if role != users.RoleAdmin {
				return fiber.ErrForbidden
			}
			allowed, err := repository.HasClassManagementAccess(c.UserContext(), userID, classID)
			return allowIf(allowed, err)
		})
	}
}

func CourseAccess(repository classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			if role == users.RoleSuperAdmin || role == users.RoleAdmin {
				return nil
			}
			courseID, err := paramID(c, "courseID")
			if err != nil {
				return err
			}
			allowed, err := repository.HasCourseAccess(c.UserContext(), userID, courseID)
			return allowIf(allowed, err)
		})
	}
}

func CourseWriteAccess(repository classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			courseID, err := paramID(c, "courseID")
			if err != nil {
				return err
			}
			if role == users.RoleSuperAdmin || role == users.RoleAdmin {
				return nil
			}
			if role != users.RoleTeacher {
				return fiber.ErrForbidden
			}
			allowed, err := repository.HasCourseWriteAccess(c.UserContext(), userID, courseID)
			return allowIf(allowed, err)
		})
	}
}

func StudentSubjectAccess(repository classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			studentID, err := paramID(c, "studentID")
			if err != nil {
				return err
			}
			if studentID == userID {
				return nil
			}
			allowed, err := repository.HasStudentAccess(c.UserContext(), userID, studentID)
			return allowIf(allowed, err)
		})
	}
}

func TeacherSubjectAccess(repository classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			teacherID, err := paramID(c, "teacherID")
			if err != nil {
				return err
			}
			if teacherID == userID {
				return nil
			}
			if role == users.RoleTeacher || role == users.RoleStudent {
				return fiber.ErrForbidden
			}
			allowed, err := repository.HasTeacherSubjectAccess(c.UserContext(), userID, teacherID)
			return allowIf(allowed, err)
		})
	}
}

func QuizScopeAccess(scope ContentScope) fiber.Handler {
	return func(c *fiber.Ctx) error {
		quizID, err := paramID(c, "quizID")
		if err != nil {
			return err
		}
		classID, courseID, err := scope.QuizScope(c.UserContext(), quizID)
		if err != nil {
			return writeAccessError(err)
		}
		classParam, err := paramID(c, "classID")
		if err != nil {
			return err
		}
		if classParam != uuid.Nil && classParam != classID {
			return fiber.ErrNotFound
		}
		courseParam, err := paramID(c, "courseID")
		if err != nil {
			return err
		}
		if courseParam != uuid.Nil && courseParam != courseID {
			return fiber.ErrNotFound
		}
		return c.Next()
	}
}

func QuestionScopeAccess(scope ContentScope) fiber.Handler {
	return func(c *fiber.Ctx) error {
		questionID, err := paramID(c, "questionID")
		if err != nil {
			return err
		}
		quizID, err := scope.QuestionQuiz(c.UserContext(), questionID)
		if err != nil {
			return writeAccessError(err)
		}
		quizParam, err := paramID(c, "quizID")
		if err != nil {
			return err
		}
		if quizParam != uuid.Nil && quizParam != quizID {
			return fiber.ErrNotFound
		}
		return c.Next()
	}
}

func OptionScopeAccess(scope ContentScope) fiber.Handler {
	return func(c *fiber.Ctx) error {
		optionID, err := paramID(c, "optionID")
		if err != nil {
			return err
		}
		questionID, err := scope.OptionQuestion(c.UserContext(), optionID)
		if err != nil {
			return writeAccessError(err)
		}
		questionParam, err := paramID(c, "questionID")
		if err != nil {
			return err
		}
		if questionParam != uuid.Nil && questionParam != questionID {
			return fiber.ErrNotFound
		}
		return c.Next()
	}
}

// ResourceWriteAccess, sınıf ID'si URL'de bulunmayan kayıtlar için kaydın
// bağlı olduğu sınıfı çözümleyip sınıf yazma yetkisini yeniden uygular.
func ResourceWriteAccess(scope ResourceScope, resolver ClassResolver, resource string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			id, err := paramID(c, resourceIDParam(resource))
			if err != nil {
				return err
			}
			if id == uuid.Nil {
				return fiber.ErrNotFound
			}
			classID, err := resolver.ClassIDForResource(c.UserContext(), resource, id)
			if err != nil {
				return writeAccessError(err)
			}
			if role == users.RoleSuperAdmin {
				return nil
			}
			if role == users.RoleAdmin {
				allowed, err := scope.HasAccess(c.UserContext(), userID, classID)
				return allowIf(allowed, err)
			}
			if role != users.RoleTeacher {
				return fiber.ErrForbidden
			}
			allowed, err := scope.HasTeacherAccess(c.UserContext(), userID, classID)
			return allowIf(allowed, err)
		})
	}
}

// ResourceReadAccess, kaydın bağlı olduğu sınıfa okuma yetkisini uygular ve
// öğrencinin yalnızca kendi kayıtlarını görmesine izin verir.
func ResourceReadAccess(scope ResourceScope, resolver ClassResolver, resource string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			id, err := paramID(c, resourceIDParam(resource))
			if err != nil {
				return err
			}
			if id == uuid.Nil {
				return fiber.ErrNotFound
			}
			ownerID, err := resolver.OwnerIDForResource(c.UserContext(), resource, id)
			if err != nil {
				return writeAccessError(err)
			}
			if role == users.RoleStudent {
				// Ders programı satırının "sahibi" öğretmendir; öğrenci kayıtlı olduğu sınıfın programını okuyabilir.
				if resource == "schedule" {
					classID, err := resolver.ClassIDForResource(c.UserContext(), resource, id)
					if err != nil {
						return writeAccessError(err)
					}
					allowed, err := scope.HasAccess(c.UserContext(), userID, classID)
					return allowIf(allowed, err)
				}
				if ownerID != userID {
					return fiber.ErrForbidden
				}
				return nil
			}
			classID, err := resolver.ClassIDForResource(c.UserContext(), resource, id)
			if err != nil {
				return writeAccessError(err)
			}
			if role == users.RoleSuperAdmin {
				return nil
			}
			if role == users.RoleAdmin {
				allowed, err := scope.HasAccess(c.UserContext(), userID, classID)
				return allowIf(allowed, err)
			}
			if role != users.RoleTeacher {
				return fiber.ErrForbidden
			}
			if ownerID == userID {
				return nil
			}
			allowed, err := scope.HasTeacherAccess(c.UserContext(), userID, classID)
			return allowIf(allowed, err)
		})
	}
}

// AttemptWriteAccess, öğrencinin yalnızca kendi sınav denemesine yazmasına izin
// verir; personel tarafında sınıf yazma yetkisi arar.
func AttemptWriteAccess(scope ResourceScope, resolver ClassResolver) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			attemptID, err := paramID(c, "attemptID")
			if err != nil {
				return err
			}
			if attemptID == uuid.Nil {
				return fiber.ErrNotFound
			}
			ownerID, err := resolver.OwnerIDForResource(c.UserContext(), "attempt", attemptID)
			if err != nil {
				return writeAccessError(err)
			}
			if role == users.RoleStudent {
				if ownerID != userID {
					return fiber.ErrForbidden
				}
				return nil
			}
			classID, err := resolver.ClassIDForResource(c.UserContext(), "attempt", attemptID)
			if err != nil {
				return writeAccessError(err)
			}
			if role == users.RoleSuperAdmin {
				return nil
			}
			if role == users.RoleAdmin {
				allowed, err := scope.HasAccess(c.UserContext(), userID, classID)
				return allowIf(allowed, err)
			}
			if role != users.RoleTeacher {
				return fiber.ErrForbidden
			}
			allowed, err := scope.HasTeacherAccess(c.UserContext(), userID, classID)
			return allowIf(allowed, err)
		})
	}
}

// AttemptStartAccess, sınav denemesi başlatmada öğrencinin kaydı servis
// tarafından denetlenirken personelin sınavın sınıfına erişimini zorunlu kılar.
func AttemptStartAccess(scope ContentScope, access classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			quizID, err := paramID(c, "quizID")
			if err != nil {
				return err
			}
			if role == users.RoleStudent {
				return nil
			}
			classID, _, err := scope.QuizScope(c.UserContext(), quizID)
			if err != nil {
				return writeAccessError(err)
			}
			if role == users.RoleSuperAdmin {
				return nil
			}
			if role == users.RoleAdmin {
				allowed, err := access.HasClassManagementAccess(c.UserContext(), userID, classID)
				return allowIf(allowed, err)
			}
			if role != users.RoleTeacher {
				return fiber.ErrForbidden
			}
			allowed, err := access.HasTeacherAccess(c.UserContext(), userID, classID)
			return allowIf(allowed, err)
		})
	}
}

// listScopeKey, global liste uçlarında çağıranın görmesi gereken sınıf kümesini
// handler'a taşır.
const listScopeKey = "list_scope_class_ids"

// ScopedClassIDs, liste handler'ının uygulayacağı sınıf kısıtını döner.
func ScopedClassIDs(c *fiber.Ctx) ([]uuid.UUID, bool) {
	ids, ok := c.Locals(listScopeKey).([]uuid.UUID)
	return ids, ok
}

// ListScopeAccess, verilen sınıfı doğrulayan, aksi halde çağıranın sınıf kümesini
// liste handler'ına enjekte eden global liste sızıntısını kapatır.
func ListScopeAccess(access classes.AccessRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return withIdentity(c, func(userID uuid.UUID, role users.Role) error {
			if role == users.RoleSuperAdmin {
				return nil
			}

			raw := c.Query("class_id")
			if raw == "" {
				raw = c.Params("classID")
			}

			if raw != "" {
				classID, err := uuid.Parse(raw)
				if err != nil {
					return core.ErrValidation
				}
				var allowed bool
				switch role {
				case users.RoleAdmin:
					allowed, err = access.HasClassManagementAccess(c.UserContext(), userID, classID)
				case users.RoleTeacher:
					allowed, err = access.HasTeacherAccess(c.UserContext(), userID, classID)
				case users.RoleStudent:
					allowed, err = access.HasAccess(c.UserContext(), userID, classID) // yalnızca kayıtlı olduğu sınıf
				default:
					return fiber.ErrForbidden
				}
				return allowIf(allowed, err)
			}
			ids, err := access.ScopedClassIDs(c.UserContext(), userID)
			if err != nil {
				return writeAccessError(err)
			}
			c.Locals(listScopeKey, ids)
			return nil
		})
	}
}

// CourseParentAccess, course altındaki nested kaynakların gerçekten o derse
// ait olduğunu doğrulayarak sınıf/sınıf öğretmeni erişimini yeterli kılar.
func CourseParentAccess(scope ContentScope) fiber.Handler {
	return func(c *fiber.Ctx) error {
		courseID, err := paramID(c, "courseID")
		if err != nil {
			return err
		}
		if courseID == uuid.Nil {
			return c.Next()
		}
		topicID, err := paramID(c, "topicID")
		if err != nil {
			return err
		}
		if topicID != uuid.Nil {
			actual, err := scope.TopicCourse(c.UserContext(), topicID)
			if err != nil {
				return writeAccessError(err)
			}
			if actual != courseID {
				return fiber.ErrNotFound
			}
			return c.Next()
		}
		lessonID, err := paramID(c, "lessonID")
		if err != nil {
			return err
		}
		if lessonID != uuid.Nil {
			actual, err := scope.CourseIDForLesson(c.UserContext(), lessonID)
			if err != nil {
				return writeAccessError(err)
			}
			if actual != courseID {
				return fiber.ErrNotFound
			}
		}
		return c.Next()
	}
}

func resourceIDParam(resource string) string {
	switch resource {
	case "attendance":
		return "attendanceID"
	case "grade":
		return "gradeID"
	case "schedule":
		return "scheduleID"
	case "attempt":
		return "attemptID"
	case "note":
		return "noteID"
	default:
		return "id"
	}
}

func withIdentity(c *fiber.Ctx, run func(userID uuid.UUID, role users.Role) error) error {
	userID, ok := c.Locals(UserIDLocal).(uuid.UUID)
	if !ok {
		return fiber.ErrUnauthorized
	}
	role, ok := c.Locals(RoleLocal).(users.Role)
	if !ok {
		return fiber.ErrUnauthorized
	}
	if err := run(userID, role); err != nil {
		if fiberError, ok := err.(*fiber.Error); ok {
			return fiberError
		}
		return err
	}
	return c.Next()
}

func allowIf(allowed bool, err error) error {
	if err != nil {
		return fiber.ErrInternalServerError
	}
	if !allowed {
		return fiber.ErrForbidden
	}
	return nil
}

func paramID(c *fiber.Ctx, name string) (uuid.UUID, error) {
	value := c.Params(name)
	if value == "" {
		return uuid.Nil, nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fiber.ErrNotFound
	}
	return parsed, nil
}

func writeAccessError(err error) error {
	if err == core.ErrNotFound || err == core.ErrValidation {
		return fiber.ErrNotFound
	}
	return fiber.ErrInternalServerError
}
