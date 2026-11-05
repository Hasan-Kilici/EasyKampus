package routes

import (
	"github.com/gofiber/fiber/v2"

	"lms/backend/internal/api/handlers"
	apiMiddleware "lms/backend/internal/api/middleware"
	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/users"
)

type Handlers struct {
	Auth        *handlers.AuthHandler
	User        *handlers.UserHandler
	School      *handlers.SchoolHandler
	Campus      *handlers.CampusHandler
	Course      *handlers.CourseHandler
	Topic       *handlers.TopicHandler
	Class       *handlers.ClassHandler
	Assignment  *handlers.AssignmentHandler
	Schedule    *handlers.ScheduleHandler
	Enrollment  *handlers.EnrollmentHandler
	Student     *handlers.StudentHandler
	Lesson      *handlers.LessonHandler
	Quiz        *handlers.QuizHandler
	Question    *handlers.QuestionHandler
	Option      *handlers.OptionHandler
	Attempt     *handlers.AttemptHandler
	Exam        *handlers.ExamHandler
	Attendance  *handlers.AttendanceHandler
	Grade       *handlers.GradeHandler
	Note        *handlers.NoteHandler
	Activity    *handlers.ActivityHandler
	Performance *handlers.PerformanceHandler
	Term        *handlers.TermHandler
	Video       *handlers.VideoHandler
}

type Dependencies struct {
	Access          classes.AccessRepository
	ContentScope    apiMiddleware.ContentScope
	ResourceScope   apiMiddleware.ClassResolver
	Auth            fiber.Handler
	LoginRate       fiber.Handler
	PerformanceRate fiber.Handler
}

func Register(app *fiber.App, handler Handlers, deps Dependencies) {
	api := app.Group("/api")
	auth := deps.Auth
	access := deps.Access
	scope := deps.ContentScope
	resource := deps.ResourceScope
	attendanceWrite := apiMiddleware.ResourceWriteAccess(access, resource, "attendance")
	gradeWrite := apiMiddleware.ResourceWriteAccess(access, resource, "grade")
	attemptWrite := apiMiddleware.AttemptWriteAccess(access, resource)
	scheduleWrite := apiMiddleware.ResourceWriteAccess(access, resource, "schedule")
	noteWrite := apiMiddleware.ResourceWriteAccess(access, resource, "note")
	attemptRead := apiMiddleware.ResourceReadAccess(access, resource, "attempt")
	scheduleRead := apiMiddleware.ResourceReadAccess(access, resource, "schedule")
	noteRead := apiMiddleware.ResourceReadAccess(access, resource, "note")
	staffOrTeacher := apiMiddleware.RequireRoles(users.RoleAdmin, users.RoleSuperAdmin, users.RoleTeacher)

	authRoutes := api.Group("/auth")
	authRoutes.Post("/register", handler.Auth.Register)
	authRoutes.Post("/login", deps.LoginRate, handler.Auth.Login)
	authRoutes.Post("/logout", auth, handler.Auth.Logout)

	usersRoutes := api.Group("/users", auth)
	usersRoutes.Get("/me", handler.User.Me)
	usersRoutes.Get("/", apiMiddleware.RequireRoles(users.RoleAdmin, users.RoleSuperAdmin), handler.User.List)
	usersRoutes.Post("/", apiMiddleware.RequireRoles(users.RoleAdmin, users.RoleSuperAdmin), handler.User.Create)
	usersRoutes.Get("/:userID", apiMiddleware.RequireRoles(users.RoleAdmin, users.RoleSuperAdmin), handler.User.Get)
	usersRoutes.Put("/:userID", apiMiddleware.RequireRoles(users.RoleAdmin, users.RoleSuperAdmin), handler.User.Update)
	usersRoutes.Delete("/:userID", apiMiddleware.RequireRoles(users.RoleAdmin, users.RoleSuperAdmin), handler.User.Delete)

	schoolsRoutes := api.Group("/schools", auth)
	schoolsRoutes.Get("/", handler.School.List)
	schoolsRoutes.Post("/", apiMiddleware.RequireRoles(users.RoleSuperAdmin), handler.School.Create)
	schoolsRoutes.Post("/start", apiMiddleware.RequireRoles(users.RoleSuperAdmin), handler.School.Start)
	schoolsRoutes.Get("/:schoolID/terms", apiMiddleware.SchoolAccess(access), handler.Term.List)
	schoolsRoutes.Post("/:schoolID/terms", apiMiddleware.SchoolWriteAccess(access), handler.Term.Start)
	schoolsRoutes.Post("/:schoolID/terms/:termID/close", apiMiddleware.SchoolWriteAccess(access), handler.Term.Close)
	schoolsRoutes.Get("/:schoolID", apiMiddleware.SchoolAccess(access), handler.School.Get)
	schoolsRoutes.Put("/:schoolID", apiMiddleware.SchoolWriteAccess(access), handler.School.Update)
	schoolsRoutes.Delete("/:schoolID", apiMiddleware.RequireRoles(users.RoleSuperAdmin), handler.School.Delete)
	schoolsRoutes.Get("/:schoolID/staff", apiMiddleware.SchoolAccess(access), handler.School.ListStaff)
	schoolsRoutes.Post("/:schoolID/staff", apiMiddleware.SchoolWriteAccess(access), handler.School.AssignStaff)
	schoolsRoutes.Delete("/:schoolID/staff/:userID", apiMiddleware.SchoolWriteAccess(access), handler.School.RemoveStaff)

	campusesRoutes := api.Group("/campuses", auth)
	campusesRoutes.Get("/", handler.Campus.List)
	campusesRoutes.Post("/", apiMiddleware.RequireRoles(users.RoleSuperAdmin), handler.Campus.Create)
	campusesRoutes.Get("/:campusID", apiMiddleware.CampusAccess(access), handler.Campus.Get)
	campusesRoutes.Put("/:campusID", apiMiddleware.CampusWriteAccess(access), handler.Campus.Update)
	campusesRoutes.Delete("/:campusID", apiMiddleware.RequireRoles(users.RoleSuperAdmin), handler.Campus.Delete)

	coursesRoutes := api.Group("/courses", auth)
	coursesRoutes.Get("/", handler.Course.List)
	coursesRoutes.Post("/", apiMiddleware.RequireRoles(users.RoleAdmin, users.RoleSuperAdmin), handler.Course.Create)
	coursesRoutes.Get("/:courseID", apiMiddleware.CourseAccess(access), handler.Course.Get)
	coursesRoutes.Put("/:courseID", apiMiddleware.CourseWriteAccess(access), handler.Course.Update)
	coursesRoutes.Delete("/:courseID", apiMiddleware.RequireRoles(users.RoleAdmin, users.RoleSuperAdmin), handler.Course.Delete)

	courseScoped := coursesRoutes.Group("/:courseID", apiMiddleware.CourseAccess(access), apiMiddleware.CourseParentAccess(scope))
	courseScoped.Get("/workload", handler.Topic.Workload)
	courseScoped.Get("/content-analysis", handler.Topic.ContentAnalysis)
	courseScoped.Get("/topics", handler.Topic.List)
	courseScoped.Post("/topics", apiMiddleware.CourseWriteAccess(access), handler.Topic.Create)
	courseScoped.Get("/topics/:topicID", handler.Topic.Get)
	courseScoped.Put("/topics/:topicID", apiMiddleware.CourseWriteAccess(access), handler.Topic.Update)
	courseScoped.Delete("/topics/:topicID", apiMiddleware.CourseWriteAccess(access), handler.Topic.Delete)
	courseScoped.Get("/lessons", handler.Lesson.List)
	courseScoped.Post("/lessons", apiMiddleware.CourseWriteAccess(access), handler.Lesson.Create)
	courseScoped.Get("/lessons/:lessonID", handler.Lesson.Get)
	courseScoped.Put("/lessons/:lessonID", apiMiddleware.CourseWriteAccess(access), handler.Lesson.Update)
	courseScoped.Delete("/lessons/:lessonID", apiMiddleware.CourseWriteAccess(access), handler.Lesson.Delete)
	courseScoped.Post("/lessons/:lessonID/review", apiMiddleware.CourseWriteAccess(access), handler.Lesson.Review)
	courseScoped.Post("/lessons/:lessonID/view", apiMiddleware.RequireRoles(users.RoleStudent), handler.Lesson.View)
	courseScoped.Post("/lessons/:lessonID/complete", apiMiddleware.RequireRoles(users.RoleStudent), handler.Lesson.Complete)
	courseScoped.Get("/lessons/:lessonID/videos", handler.Video.List)
	courseScoped.Post("/lessons/:lessonID/videos", apiMiddleware.CourseWriteAccess(access), handler.Video.Create)
	courseScoped.Post("/lessons/:lessonID/videos/presign", apiMiddleware.CourseWriteAccess(access), handler.Video.Presign)
	courseScoped.Post("/lessons/:lessonID/videos/completions", apiMiddleware.RequireRoles(users.RoleStudent), handler.Video.Complete)
	courseScoped.Get("/lessons/:lessonID/videos/completions", handler.Video.ListCompletions)
	courseScoped.Get("/lessons/:lessonID/videos/:videoID", handler.Video.Get)
	courseScoped.Get("/lessons/:lessonID/videos/:videoID/play", handler.Video.Play)
	courseScoped.Put("/lessons/:lessonID/videos/:videoID", apiMiddleware.CourseWriteAccess(access), handler.Video.Update)
	courseScoped.Delete("/lessons/:lessonID/videos/:videoID", apiMiddleware.CourseWriteAccess(access), handler.Video.Delete)

	classesRoutes := api.Group("/classes", auth)
	classesRoutes.Get("/", handler.Class.List)
	classesRoutes.Post("/", apiMiddleware.RequireRoles(users.RoleAdmin, users.RoleSuperAdmin), handler.Class.Create)
	classesRoutes.Get("/:classID", apiMiddleware.ClassAccess(access), handler.Class.Get)
	classesRoutes.Put("/:classID", apiMiddleware.ClassManagementAccess(access), handler.Class.Update)
	classesRoutes.Delete("/:classID", apiMiddleware.ClassManagementAccess(access), handler.Class.Delete)
	classesRoutes.Get("/:classID/schedules", apiMiddleware.ClassAccess(access), handler.Schedule.List)
	classesRoutes.Post("/:classID/schedules", apiMiddleware.ClassWriteAccess(access), handler.Schedule.Create)

	classScoped := classesRoutes.Group("/:classID", apiMiddleware.ClassAccess(access))
	classScoped.Get("/courses", handler.Assignment.ListCourses)
	classScoped.Post("/courses", apiMiddleware.ClassManagementAccess(access), handler.Assignment.AssignCourse)
	classScoped.Delete("/courses/:courseID", apiMiddleware.ClassManagementAccess(access), handler.Assignment.RemoveCourse)
	classScoped.Get("/teachers", handler.Assignment.ListTeachers)
	classScoped.Post("/teachers", apiMiddleware.ClassManagementAccess(access), handler.Assignment.AssignTeacher)
	classScoped.Delete("/teachers/:teacherID", apiMiddleware.ClassManagementAccess(access), handler.Assignment.RemoveTeacher)
	classScoped.Get("/enrollments", staffOrTeacher, handler.Enrollment.ListByClass)
	classScoped.Post("/enrollments", apiMiddleware.ClassManagementAccess(access), handler.Enrollment.Enroll)
	classScoped.Post("/enrollments/bulk", apiMiddleware.ClassManagementAccess(access), handler.Enrollment.EnrollBulk)
	classScoped.Post("/transfers", apiMiddleware.ClassManagementAccess(access), handler.Enrollment.Transfer)
	classScoped.Delete("/enrollments/:studentID", apiMiddleware.ClassManagementAccess(access), handler.Enrollment.Withdraw)
	classScoped.Get("/quizzes", handler.Quiz.List)
	classScoped.Post("/quizzes", apiMiddleware.ClassWriteAccess(access), handler.Quiz.Create)
	classScoped.Get("/attendance", handler.Attendance.List)
	classScoped.Post("/attendance", apiMiddleware.ClassWriteAccess(access), handler.Attendance.Record)
	classScoped.Get("/grades", handler.Grade.List)
	classScoped.Post("/grades", apiMiddleware.ClassWriteAccess(access), handler.Grade.Record)
	classScoped.Post("/grades/bulk", apiMiddleware.ClassWriteAccess(access), handler.Grade.RecordBulk)
	classScoped.Get("/notes", handler.Note.List)
	classScoped.Post("/notes", apiMiddleware.ClassWriteAccess(access), handler.Note.Create)
	classScoped.Get("/attempts", staffOrTeacher, handler.Attempt.List)

	// Sınav iş akışı: sorular/seçenekler sınavla birlikte atomik yazılır; öğrenciler bu uçlara erişemez.
	classScoped.Post("/exams", staffOrTeacher, apiMiddleware.ClassWriteAccess(access), handler.Exam.Compose)
	examScoped := classScoped.Group("/exams/:quizID", staffOrTeacher, apiMiddleware.QuizScopeAccess(scope), apiMiddleware.CourseParentAccess(scope))
	examScoped.Get("/", handler.Exam.Get)
	examScoped.Put("/", apiMiddleware.ClassWriteAccess(access), handler.Exam.Replace)
	examScoped.Post("/review", apiMiddleware.ClassWriteAccess(access), handler.Exam.Review)
	examScoped.Get("/results", handler.Exam.Results)

	quizScoped := classScoped.Group("/quizzes/:quizID", apiMiddleware.QuizScopeAccess(scope), apiMiddleware.CourseParentAccess(scope))
	quizScoped.Get("/", handler.Quiz.Get)
	quizScoped.Put("/", apiMiddleware.ClassWriteAccess(access), handler.Quiz.Update)
	quizScoped.Delete("/", apiMiddleware.ClassWriteAccess(access), handler.Quiz.Delete)
	quizScoped.Post("/review", apiMiddleware.ClassWriteAccess(access), handler.Quiz.Review)
	quizScoped.Get("/questions", handler.Question.List)
	quizScoped.Post("/questions", apiMiddleware.ClassWriteAccess(access), handler.Question.Create)

	questionScoped := quizScoped.Group("/questions/:questionID", apiMiddleware.QuestionScopeAccess(scope))
	questionScoped.Get("/", handler.Question.Get)
	questionScoped.Put("/", apiMiddleware.ClassWriteAccess(access), handler.Question.Update)
	questionScoped.Delete("/", apiMiddleware.ClassWriteAccess(access), handler.Question.Delete)
	questionScoped.Post("/review", apiMiddleware.ClassWriteAccess(access), handler.Question.Review)
	questionScoped.Get("/options", handler.Option.List)
	questionScoped.Post("/options", apiMiddleware.ClassWriteAccess(access), handler.Option.Create)

	optionScoped := questionScoped.Group("/options/:optionID", apiMiddleware.OptionScopeAccess(scope))
	optionScoped.Get("/", handler.Option.Get)
	optionScoped.Put("/", apiMiddleware.ClassWriteAccess(access), handler.Option.Update)
	optionScoped.Delete("/", apiMiddleware.ClassWriteAccess(access), handler.Option.Delete)

	schedulesRoutes := api.Group("/schedules", auth)
	schedulesRoutes.Get("/", apiMiddleware.ListScopeAccess(access), handler.Schedule.List)
	schedulesRoutes.Get("/:scheduleID", scheduleRead, handler.Schedule.Get)
	schedulesRoutes.Put("/:scheduleID", scheduleWrite, handler.Schedule.Update)
	schedulesRoutes.Delete("/:scheduleID", scheduleWrite, handler.Schedule.Delete)

	studentsRoutes := api.Group("/students", auth)
	studentsRoutes.Get("/me/enrollments", handler.Enrollment.ListByStudent)
	studentsRoutes.Get("/me/lessons", handler.Student.Lessons)
	studentsRoutes.Get("/me/activities", handler.Activity.List)
	studentsRoutes.Get("/me/activities/daily", handler.Activity.Daily)
	studentsRoutes.Post("/me/activities", handler.Activity.Record)
	studentsRoutes.Get("/me/performance", deps.PerformanceRate, handler.Performance.Student)
	studentsRoutes.Get("/:studentID/enrollments", apiMiddleware.StudentSubjectAccess(access), handler.Enrollment.ListByStudent)
	studentsRoutes.Get("/:studentID/profile", apiMiddleware.StudentSubjectAccess(access), handler.Student.Profile)
	studentsRoutes.Get("/:studentID/lessons", apiMiddleware.StudentSubjectAccess(access), handler.Student.Lessons)
	studentsRoutes.Get("/:studentID/activities", apiMiddleware.StudentSubjectAccess(access), handler.Activity.List)
	studentsRoutes.Get("/:studentID/activities/daily", apiMiddleware.StudentSubjectAccess(access), handler.Activity.Daily)
	studentsRoutes.Get("/:studentID/attendance", apiMiddleware.StudentSubjectAccess(access), handler.Attendance.List)
	studentsRoutes.Get("/:studentID/grades", apiMiddleware.StudentSubjectAccess(access), handler.Grade.List)
	studentsRoutes.Get("/:studentID/notes", apiMiddleware.StudentSubjectAccess(access), handler.Note.List)
	studentsRoutes.Get("/:studentID/performance", apiMiddleware.StudentSubjectAccess(access), deps.PerformanceRate, handler.Performance.Student)

	teachersRoutes := api.Group("/teachers", auth)
	teachersRoutes.Get("/me/performance", deps.PerformanceRate, handler.Performance.Teacher)
	teachersRoutes.Get("/:teacherID/performance", apiMiddleware.TeacherSubjectAccess(access), deps.PerformanceRate, handler.Performance.Teacher)

	attendanceRoutes := api.Group("/attendance", auth)
	attendanceRoutes.Get("/", apiMiddleware.ListScopeAccess(access), handler.Attendance.List)
	attendanceRoutes.Get("/summary", apiMiddleware.ListScopeAccess(access), staffOrTeacher, handler.Attendance.Summary)
	attendanceRoutes.Put("/:attendanceID", attendanceWrite, handler.Attendance.Update)
	attendanceRoutes.Delete("/:attendanceID", attendanceWrite, handler.Attendance.Delete)

	gradesRoutes := api.Group("/grades", auth)
	gradesRoutes.Get("/export", apiMiddleware.ListScopeAccess(access), staffOrTeacher, handler.Grade.Export)
	gradesRoutes.Post("/import", gradeWrite, handler.Grade.Import)
	gradesRoutes.Get("/:gradeID/audit", gradeWrite, handler.Grade.Audit)
	gradesRoutes.Get("/", apiMiddleware.ListScopeAccess(access), handler.Grade.List)
	gradesRoutes.Put("/:gradeID", gradeWrite, handler.Grade.Update)
	gradesRoutes.Delete("/:gradeID", gradeWrite, handler.Grade.Delete)

	notesRoutes := api.Group("/notes", auth)
	notesRoutes.Get("/", apiMiddleware.ListScopeAccess(access), handler.Note.List)
	notesRoutes.Get("/:noteID", noteRead, handler.Note.Get)
	notesRoutes.Put("/:noteID", noteWrite, handler.Note.Update)
	notesRoutes.Delete("/:noteID", noteWrite, handler.Note.Delete)

	attemptsRoutes := api.Group("/attempts", auth)
	attemptsRoutes.Get("/", apiMiddleware.ListScopeAccess(access), handler.Attempt.List)
	attemptsRoutes.Post("/quizzes/:quizID/start", apiMiddleware.QuizScopeAccess(scope), apiMiddleware.AttemptStartAccess(scope, access), apiMiddleware.RequireRoles(users.RoleStudent, users.RoleTeacher, users.RoleAdmin, users.RoleSuperAdmin), handler.Attempt.Start)
	attemptsRoutes.Get("/:attemptID", attemptRead, handler.Attempt.Get)
	attemptsRoutes.Get("/:attemptID/exam", attemptRead, handler.Exam.Take)
	attemptsRoutes.Get("/:attemptID/result", attemptRead, handler.Exam.Result)
	attemptsRoutes.Put("/:attemptID/answers", attemptWrite, handler.Attempt.Answer)
	attemptsRoutes.Post("/:attemptID/submit", attemptWrite, handler.Attempt.Submit)
}
