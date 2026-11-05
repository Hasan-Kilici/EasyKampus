package handlers

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"lms/backend/internal/storage"
	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/assignments"
	"lms/backend/pkg/domain/classes"
	"lms/backend/pkg/domain/enrollments"
	"lms/backend/pkg/domain/lessons"
	"lms/backend/pkg/domain/users"
	"lms/backend/pkg/domain/video"
)

type VideoHandler struct {
	service     video.Service
	lessons     lessons.Service
	enrollments enrollments.Service
	assignments assignments.Repository
	access      classes.AccessRepository
	storage     storage.R2Config
}

func NewVideoHandler(service video.Service, lessons lessons.Service, assignments assignments.Repository, enrollments enrollments.Service, access classes.AccessRepository, storageConfig storage.R2Config) *VideoHandler {
	return &VideoHandler{service: service, lessons: lessons, assignments: assignments, enrollments: enrollments, access: access, storage: storageConfig}
}

func (h *VideoHandler) canViewLesson(c *fiber.Ctx, courseID string, lessonID string) error {
	userID, role, err := identity(c)
	if err != nil {
		return err
	}
	if role == users.RoleSuperAdmin || role == users.RoleAdmin {
		return nil
	}
	if role == users.RoleTeacher {
		return nil
	}
	if role == users.RoleStudent {
		ens, err := h.enrollments.ListByStudent(c.UserContext(), userID.String())
		if err != nil {
			return writeDomainError(c, err)
		}
		cid, err := uuid.Parse(courseID)
		if err != nil {
			return badRequest(c, err)
		}
		for _, e := range ens {
			if e.Status != enrollments.StatusActive {
				continue
			}
			if e.CourseID != nil && *e.CourseID == cid {
				return nil
			}
			if ok, err := h.assignments.ClassCourseExists(c.UserContext(), e.ClassID, cid); err == nil && ok {
				return nil
			}
		}
		return core.ErrForbidden
	}
	return core.ErrForbidden
}

// @Summary	List lesson videos
// @Tags		Video
// @Produce	json
// @Param		courseID	path		string					true	"Ders kimliği (UUID)"
// @Param		lessonID	path		string					true	"Ders içeriği kimliği (UUID)"
// @Success	200			{array}		video.Video				"successful response"
// @Failure	401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure	403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure	404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Security	BearerAuth
// @Router		/api/courses/{courseID}/lessons/{lessonID}/videos [GET]
func (h *VideoHandler) List(c *fiber.Ctx) error {
	courseID := c.Params("courseID")
	lessonID := c.Params("lessonID")
	if err := h.canViewLesson(c, courseID, lessonID); err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	lid, err := uuid.Parse(lessonID)
	if err != nil {
		return badRequest(c, err)
	}
	res, err := h.service.ListVideosByLesson(c.UserContext(), video.ListParams{Page: page, Limit: limit, LessonID: lid})
	if err != nil {
		return writeDomainError(c, err)
	}
	return list(c, res, int64(len(res)))
}

// @Summary	Create video metadata
// @Tags		Video
// @Produce	json
// @Accept		json
// @Param		courseID	path		string					true	"Ders kimliği (UUID)"
// @Param		lessonID	path		string					true	"Ders içeriği kimliği (UUID)"
// @Param		request		body		video.Video				true	"İstek gövdesi"
// @Success	201			{object}	video.Video				"successful response"
// @Failure	400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure	401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure	403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure	422			{object}	handlers.ErrorResponse	"doğrulama hatası"
// @Security	BearerAuth
// @Router		/api/courses/{courseID}/lessons/{lessonID}/videos [POST]
func (h *VideoHandler) Create(c *fiber.Ctx) error {
	var input video.Video
	if err := c.BodyParser(&input); err != nil {
		return badRequest(c, err)
	}
	lid, err := uuid.Parse(c.Params("lessonID"))
	if err != nil {
		return badRequest(c, err)
	}
	input.LessonID = lid
	uid, _, err := identity(c)
	if err != nil {
		return err
	}
	input.CreatedByID = uid
	if input.Title == "" || input.URL == "" {
		return badRequest(c, core.ErrValidation)
	}
	if err := h.service.CreateVideo(c.UserContext(), &input); err != nil {
		return writeDomainError(c, err)
	}
	return created(c, input)
}

// @Summary	Get presigned upload URL
// @Tags		Video
// @Produce	json
// @Accept		json
// @Param		courseID	path		string					true	"Ders kimliği (UUID)"
// @Param		lessonID	path		string					true	"Ders içeriği kimliği (UUID)"
// @Param		request		body		object					true	"filename"
// @Success	200			{object}	map[string]string		"successful response"
// @Failure	401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure	403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Security	BearerAuth
// @Router		/api/courses/{courseID}/lessons/{lessonID}/videos/presign [POST]
func (h *VideoHandler) Presign(c *fiber.Ctx) error {
	var body map[string]string
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	filename := strings.TrimSpace(body["filename"])
	if filename == "" {
		filename = "video.mp4"
	}
	ext := strings.ToLower(filepath.Ext(filename))
	allowed := map[string]bool{".mp4": true, ".webm": true, ".mov": true, ".m4v": true, ".ogg": true}
	if !allowed[ext] {
		return badRequest(c, core.ErrValidation)
	}
	contentType := strings.TrimSpace(body["content_type"])
	if contentType == "" {
		contentType = "video/mp4"
	}
	key := "videos/" + uuid.NewString() + ext
	if !h.storage.Configured() {
		return ok(c, fiber.Map{
			"key":        key,
			"configured": false,
			"upload_url": "",
			"public_url": "",
			"message":    "R2 is not configured on this server",
		})
	}
	uploadURL, err := h.storage.PresignPut(key, contentType, 15*time.Minute)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, fiber.Map{
		"key":          key,
		"configured":   true,
		"upload_url":   uploadURL,
		"method":       "PUT",
		"content_type": contentType,
		"expires_in":   900,
		"public_url":   h.storage.PublicURLFor(key),
	})
}

// @Summary Get video playback URL
// @Tags Video
// @Produce json
// @Param courseID path string true "Ders kimliği (UUID)"
// @Param lessonID path string true "Ders içeriği kimliği (UUID)"
// @Param videoID path string true "Video kimliği (UUID)"
// @Success 200 {object} map[string]string "successful response"
// @Failure 401 {object} handlers.ErrorResponse "kimlik doğrulanmadı"
// @Failure 403 {object} handlers.ErrorResponse "yetkisiz işlem"
// @Failure 404 {object} handlers.ErrorResponse "kayıt bulunamadı"
// @Security BearerAuth
// @Router /api/courses/{courseID}/lessons/{lessonID}/videos/{videoID}/play [GET]
func (h *VideoHandler) Play(c *fiber.Ctx) error {
	if err := h.canViewLesson(c, c.Params("courseID"), c.Params("lessonID")); err != nil {
		return err
	}
	vid, err := uuid.Parse(c.Params("videoID"))
	if err != nil {
		return badRequest(c, err)
	}
	v, err := h.service.GetVideoByID(c.UserContext(), vid)
	if err != nil {
		return writeDomainError(c, err)
	}
	key := h.storageKey(v.URL)
	if key == "" {
		return ok(c, fiber.Map{"url": v.URL, "expires_in": 0})
	}
	if public := h.storage.PublicURLFor(key); public != "" {
		return ok(c, fiber.Map{"url": public, "expires_in": 0})
	}
	if !h.storage.Configured() {
		return ok(c, fiber.Map{"url": v.URL, "expires_in": 0})
	}
	url, err := h.storage.PresignGet(key, 15*time.Minute)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, fiber.Map{"url": url, "expires_in": 900})
}

func (h *VideoHandler) storageKey(rawURL string) string {
	if strings.HasPrefix(rawURL, "r2://") {
		return strings.TrimPrefix(rawURL, "r2://")
	}
	if h.storage.PublicURL != "" && strings.HasPrefix(rawURL, h.storage.PublicURL+"/") {
		return strings.TrimPrefix(rawURL, h.storage.PublicURL+"/")
	}
	if strings.HasPrefix(rawURL, "videos/") {
		return rawURL
	}
	return ""
}

// @Summary	Get video
// @Tags		Video
// @Produce	json
// @Param		courseID	path		string					true	"Ders kimliği (UUID)"
// @Param		lessonID	path		string					true	"Ders içeriği kimliği (UUID)"
// @Param		videoID		path		string					true	"Video kimliği (UUID)"
// @Success	200			{object}	video.Video				"successful response"
// @Failure	401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure	403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure	404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Security	BearerAuth
// @Router		/api/courses/{courseID}/lessons/{lessonID}/videos/{videoID} [GET]
func (h *VideoHandler) Get(c *fiber.Ctx) error {
	if err := h.canViewLesson(c, c.Params("courseID"), c.Params("lessonID")); err != nil {
		return err
	}
	vid, err := uuid.Parse(c.Params("videoID"))
	if err != nil {
		return badRequest(c, err)
	}
	v, err := h.service.GetVideoByID(c.UserContext(), vid)
	if err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, v)
}

// @Summary	Update video
// @Tags		Video
// @Produce	json
// @Accept		json
// @Param		courseID	path		string					true	"Ders kimliği (UUID)"
// @Param		lessonID	path		string					true	"Ders içeriği kimliği (UUID)"
// @Param		videoID		path		string					true	"Video kimliği (UUID)"
// @Param		request		body		video.Video				true	"İstek gövdesi"
// @Success	200			{object}	video.Video				"successful response"
// @Failure	400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure	401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure	403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure	404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Security	BearerAuth
// @Router		/api/courses/{courseID}/lessons/{lessonID}/videos/{videoID} [PUT]
func (h *VideoHandler) Update(c *fiber.Ctx) error {
	var input video.Video
	if err := c.BodyParser(&input); err != nil {
		return badRequest(c, err)
	}
	vid, err := uuid.Parse(c.Params("videoID"))
	if err != nil {
		return badRequest(c, err)
	}
	existing, err := h.service.GetVideoByID(c.UserContext(), vid)
	if err != nil {
		return writeDomainError(c, err)
	}
	if input.Title == "" {
		input.Title = existing.Title
	}
	if input.URL == "" {
		input.URL = existing.URL
	}
	input.ID = vid
	input.LessonID = existing.LessonID
	input.CreatedByID = existing.CreatedByID
	if err := h.service.UpdateVideo(c.UserContext(), &input); err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, input)
}

// @Summary	Delete video
// @Tags		Video
// @Produce	json
// @Param		courseID	path	string	true	"Ders kimliği (UUID)"
// @Param		lessonID	path	string	true	"Ders içeriği kimliği (UUID)"
// @Param		videoID		path	string	true	"Video kimliği (UUID)"
// @Success	204			"no content"
// @Failure	401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure	403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Failure	404			{object}	handlers.ErrorResponse	"kayıt bulunamadı"
// @Security	BearerAuth
// @Router		/api/courses/{courseID}/lessons/{lessonID}/videos/{videoID} [DELETE]
func (h *VideoHandler) Delete(c *fiber.Ctx) error {
	vid, err := uuid.Parse(c.Params("videoID"))
	if err != nil {
		return badRequest(c, err)
	}
	if err := h.service.DeleteVideo(c.UserContext(), vid); err != nil {
		return writeDomainError(c, err)
	}
	return noContent(c)
}

// @Summary	Mark video complete
// @Tags		Video
// @Produce	json
// @Accept		json
// @Param		courseID	path		string					true	"Ders kimliği (UUID)"
// @Param		lessonID	path		string					true	"Ders içeriği kimliği (UUID)"
// @Param		request		body		object					true	"video_id"
// @Success	200			{object}	map[string]bool			"successful response"
// @Failure	400			{object}	handlers.ErrorResponse	"geçersiz istek"
// @Failure	401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Failure	403			{object}	handlers.ErrorResponse	"yetkisiz işlem"
// @Security	BearerAuth
// @Router		/api/courses/{courseID}/lessons/{lessonID}/videos/completions [POST]
func (h *VideoHandler) Complete(c *fiber.Ctx) error {
	var body map[string]string
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, err)
	}
	vid, err := uuid.Parse(body["video_id"])
	if err != nil {
		return badRequest(c, err)
	}
	sid, _, err := identity(c)
	if err != nil {
		return err
	}
	lid, err := uuid.Parse(c.Params("lessonID"))
	if err != nil {
		return badRequest(c, err)
	}
	v, err := h.service.GetVideoByID(c.UserContext(), vid)
	if err != nil {
		return writeDomainError(c, err)
	}
	var courseID uuid.UUID
	if l, err := h.lessons.GetByID(c.UserContext(), lid.String()); err == nil {
		courseID = l.CourseID
	} else {
		courseID = v.LessonID
	}
	comp := video.VideoCompletion{StudentID: sid, VideoID: vid, LessonID: lid, CourseID: courseID, CompletedAt: time.Now().UTC()}
	if err := h.service.UpsertVideoCompletions(c.UserContext(), []video.VideoCompletion{comp}); err != nil {
		return writeDomainError(c, err)
	}
	return ok(c, fiber.Map{"completed": true})
}

// @Summary	List video completions
// @Tags		Video
// @Produce	json
// @Param		courseID	path		string					true	"Ders kimliği (UUID)"
// @Param		lessonID	path		string					true	"Ders içeriği kimliği (UUID)"
// @Success	200			{array}		video.VideoCompletion	"successful response"
// @Failure	401			{object}	handlers.ErrorResponse	"kimlik doğrulanmadı"
// @Security	BearerAuth
// @Router		/api/courses/{courseID}/lessons/{lessonID}/videos/completions [GET]
func (h *VideoHandler) ListCompletions(c *fiber.Ctx) error {
	uid, role, err := identity(c)
	if err != nil {
		return err
	}
	var sid uuid.UUID
	if role != users.RoleStudent {
		if s := c.Query("student_id"); s != "" {
			sid, err = uuid.Parse(s)
			if err != nil {
				return badRequest(c, err)
			}
		} else {
			sid = uid
		}
	} else {
		sid = uid
	}
	var lid *uuid.UUID
	if l := c.Params("lessonID"); l != "" {
		if parsed, err := uuid.Parse(l); err == nil {
			lid = &parsed
		}
	}
	res, err := h.service.ListVideoCompletions(c.UserContext(), sid, lid)
	if err != nil {
		return writeDomainError(c, err)
	}
	return list(c, res, int64(len(res)))
}
