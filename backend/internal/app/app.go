package app

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/swagger"
	"gorm.io/gorm"

	_ "lms/backend/docs"
	"lms/backend/internal/analytics"
	"lms/backend/internal/api/handlers"
	apiMiddleware "lms/backend/internal/api/middleware"
	"lms/backend/internal/api/routes"
	"lms/backend/internal/cache"
	"lms/backend/internal/config"
	"lms/backend/internal/database"
	"lms/backend/internal/storage"
	"lms/backend/internal/workflows"
	"lms/backend/pkg/domain/activities"
	"lms/backend/pkg/domain/assignments"
	"lms/backend/pkg/domain/attempts"
	"lms/backend/pkg/domain/attendance"
	"lms/backend/pkg/domain/campuses"
	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/courses"
	"lms/backend/pkg/domain/enrollments"
	"lms/backend/pkg/domain/exams"
	"lms/backend/pkg/domain/grades"
	"lms/backend/pkg/domain/lessons"
	"lms/backend/pkg/domain/notes"
	"lms/backend/pkg/domain/options"
	"lms/backend/pkg/domain/performance"
	"lms/backend/pkg/domain/questions"
	"lms/backend/pkg/domain/quizes"
	"lms/backend/pkg/domain/schedules"
	"lms/backend/pkg/domain/schools"
	"lms/backend/pkg/domain/sessions"
	"lms/backend/pkg/domain/terms"
	"lms/backend/pkg/domain/topics"
	"lms/backend/pkg/domain/users"
	"lms/backend/pkg/domain/video"
)

func New(cfg config.Config) (*fiber.App, *gorm.DB, error) {
	db, err := database.Open(cfg)
	if err != nil {
		return nil, nil, err
	}
	if err := database.Migrate(db); err != nil {
		return nil, db, err
	}
	if err := database.EnsureSuperAdmin(context.Background(), db, cfg.SuperAdminEmail, cfg.SuperAdminPassword); err != nil {
		return nil, db, err
	}

	app := fiber.New(fiber.Config{BodyLimit: 1 * 1024 * 1024})
	app.Use(apiMiddleware.Recovery())
	app.Use(apiMiddleware.RequestID())
	app.Use(apiMiddleware.SecurityHeaders())
	app.Use(apiMiddleware.CORS(cfg.CORSOrigins))
	if cfg.RateLimit > 0 {
		app.Use(limiter.New(limiter.Config{Max: cfg.RateLimit, Expiration: cfg.RateWindow}))
	}

	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"data": fiber.Map{"status": "ok"}, "error": nil})
	})
	app.Get("/api/swagger/*", swagger.New(swagger.Config{URL: "/api/swagger/doc.json"}))

	userRepository := database.NewUserRepository(db)
	sessionRepository := database.NewSessionRepository(db)
	accessRepository := database.NewAccessRepository(db)
	scopeRepository := database.NewScopeRepository(db)

	schoolRepository := database.NewSchoolRepository(db)
	campusRepository := database.NewCampusRepository(db)
	courseRepository := database.NewCourseRepository(db)
	topicRepository := database.NewTopicRepository(db)
	classRepository := database.NewClassRepository(db, accessRepository)
	assignmentRepository := database.NewAssignmentRepository(db)
	enrollmentRepository := database.NewEnrollmentRepository(db)
	scheduleRepository := database.NewScheduleRepository(db)
	attendanceRepository := database.NewAttendanceRepository(db)
	gradeRepository := database.NewGradeRepository(db)
	noteRepository := database.NewNoteRepository(db)
	activityRepository := database.NewActivityRepository(db)
	attemptRepository := database.NewAttemptRepository(db)
	gradingRepository := database.NewGradingRepository(db)
	lessonRepository := database.NewLessonRepository(db)
	quizRepository := database.NewQuizRepository(db)
	questionRepository := database.NewQuestionRepository(db)
	optionRepository := database.NewOptionRepository(db)
	termRepository := database.NewTermRepository(db)
	performanceRepository := database.NewPerformanceRepository(db)

	contentCache, err := cache.New(cfg.CacheSize, cfg.CacheTTL)
	if err != nil {
		return nil, db, err
	}
	performanceCache, err := cache.New(cfg.PerformanceCacheSize, cfg.PerformanceCacheTTL)
	if err != nil {
		return nil, db, err
	}

	performanceService := performance.NewService(performanceRepository, database.NewPerformanceInputBuilder(db), analytics.NewClient(cfg.PerformanceServiceURL, cfg.PerformanceTimeout).WithInternalToken(cfg.PerformanceInternalToken).WithBypassSecret(cfg.PerformanceBypassSecret), performanceCache)
	enrollmentScope := database.NewEnrollmentScopeSource(enrollmentRepository)
	activityService := activities.NewService(activityRepository, performanceService)
	attendanceService := attendance.NewService(attendanceRepository, classRepository, assignmentRepository, enrollmentScope, performanceService)
	gradeService := grades.NewService(gradeRepository, classRepository, assignmentRepository, enrollmentScope, performanceService)
	noteService := notes.NewService(noteRepository, classRepository, assignmentRepository)
	attemptService := attempts.NewService(attemptRepository, database.NewAttemptQuizSource(quizRepository), gradingRepository, enrollmentScope, performanceService, activityService)
	lessonService := lessons.NewService(lessonRepository, courseRepository, performanceService, contentCache)
	lessonCompletionService := lessons.NewCompletionService(lessonRepository, courseRepository, performanceService, activityService, contentCache)
	quizService := quizes.NewService(quizRepository, classRepository, assignmentRepository, performanceService, contentCache)
	questionService := questions.NewService(questionRepository, database.NewQuestionQuizSource(quizRepository), database.NewTopicCourseSource(topicRepository), performanceService)
	optionService := options.NewService(optionRepository, scopeRepository, performanceService)
	termService := terms.NewService(termRepository)
	schoolService := schools.NewService(schoolRepository)
	campusService := campuses.NewService(campusRepository, schoolRepository)
	courseService := courses.NewService(courseRepository, schoolRepository)
	topicService := topics.NewService(topicRepository, courseRepository)
	classService := classes.NewService(classRepository, schoolRepository, campusRepository)
	assignmentService := assignments.NewService(assignmentRepository, classRepository, courseRepository)
	enrollmentService := enrollments.NewService(enrollmentRepository, classRepository, userRepository, assignmentRepository)
	scheduleService := schedules.NewService(scheduleRepository, classRepository, assignmentRepository, assignmentRepository)
	videoRepository := database.NewVideoRepository(db)
	videoService := video.NewService(videoRepository)
	storageConfig := storage.R2Config{
		AccountID:       cfg.R2AccountID,
		AccessKeyID:     cfg.R2AccessKeyID,
		SecretAccessKey: cfg.R2SecretAccessKey,
		Bucket:          cfg.R2Bucket,
		Endpoint:        cfg.R2Endpoint,
		PublicURL:       cfg.R2PublicURL,
	}

	sessionService := cache.NewSessionService(sessions.NewService(sessionRepository), cfg.SessionCacheSize, cfg.SessionCacheTTL)
	userService := users.NewService(userRepository)

	examService := exams.NewService(database.NewExamRepository(db), classRepository, assignmentRepository, database.NewTopicCourseSource(topicRepository), attemptService, performanceService, contentCache)

	routes.Register(app, routes.Handlers{
		Auth:        handlers.NewAuthHandler(userService, sessionService, workflows.NewStudentRegistrar(db), &cfg.JWTSecret, cfg.JWTExpiration, cfg.StatelessAuth),
		User:        handlers.NewUserHandler(userService),
		School:      handlers.NewSchoolHandler(schoolService, workflows.NewSchoolStarter(db)),
		Campus:      handlers.NewCampusHandler(campusService),
		Course:      handlers.NewCourseHandler(courseService),
		Topic:       handlers.NewTopicHandler(topicService, courseService),
		Class:       handlers.NewClassHandler(classService),
		Assignment:  handlers.NewAssignmentHandler(assignmentService),
		Schedule:    handlers.NewScheduleHandler(scheduleService),
		Enrollment:  handlers.NewEnrollmentHandler(enrollmentService),
		Student:     handlers.NewStudentHandler(lessonCompletionService, userService),
		Lesson:      handlers.NewLessonHandler(lessonService),
		Quiz:        handlers.NewQuizHandler(quizService),
		Question:    handlers.NewQuestionHandler(questionService),
		Option:      handlers.NewOptionHandler(optionService),
		Attempt:     handlers.NewAttemptHandler(attemptService),
		Exam:        handlers.NewExamHandler(examService, attemptService),
		Attendance:  handlers.NewAttendanceHandler(attendanceService),
		Grade:       handlers.NewGradeHandler(gradeService),
		Note:        handlers.NewNoteHandler(noteService),
		Activity:    handlers.NewActivityHandler(activityService),
		Performance: handlers.NewPerformanceHandler(performanceService),
		Term:        handlers.NewTermHandler(termService),
		Video:       handlers.NewVideoHandler(videoService, lessonService, assignmentRepository, enrollmentService, accessRepository, storageConfig),
	}, routes.Dependencies{
		Access:          accessRepository,
		ContentScope:    scopeRepository,
		ResourceScope:   scopeRepository,
		Auth:            apiMiddleware.Auth(cfg.JWTSecret, sessionService, cfg.StatelessAuth),
		LoginRate:       limiter.New(limiter.Config{Max: 10, Expiration: cfg.RateWindow}),
		PerformanceRate: limiter.New(limiter.Config{Max: cfg.PerformanceRateLimit, Expiration: cfg.RateWindow}),
	})
	return app, db, nil
}
