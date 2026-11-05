package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"lms/backend/pkg/domain/scopeguard"
)

// studentSubject, öğrenci rolünde öğrenci filtresini kendisine sabitler; başkasının
// kimliği istenirse 403 döner. Sınıf kapsamı tek başına yetmez: sınıf arkadaşlarının
// kayıtları aynı sınıfta durur.
func studentSubject(c *fiber.Ctx, requested *uuid.UUID) (*uuid.UUID, error) {
	userID, role, err := identity(c)
	if err != nil {
		return nil, err
	}
	subject, err := scopeguard.StudentSubject(role, userID, requested)
	if err != nil {
		return nil, fiber.ErrForbidden
	}
	return subject, nil
}

// sharedNotesOnly, öğrenciye yalnızca "öğrenci görebilir" notları gösterir.
func sharedNotesOnly(c *fiber.Ctx) (bool, error) {
	_, role, err := identity(c)
	if err != nil {
		return false, err
	}
	return scopeguard.SharedNotesOnly(role, queryBool(c, "shared")), nil
}

// listViewer, okul/kampüs/ders listelerini bakan kullanıcının görebildikleriyle sınırlamak için
// kimliği ve rolü taşır. Bu olmadan her kullanıcı tüm okulların kayıtlarını listeleyebilirdi.
func listViewer(c *fiber.Ctx) (*scopeguard.Viewer, error) {
	userID, role, err := identity(c)
	if err != nil {
		return nil, err
	}
	return &scopeguard.Viewer{UserID: userID, Role: role}, nil
}
