// Package scopeguard, öğrencinin yalnızca kendi kayıtlarını görmesini sağlayan
// saf kuralları içerir. Liste uçları sınıf kapsamıyla birlikte bu kuralları da
// uygulamalıdır; aksi halde sınıf kapsamı öğrenciye sınıf arkadaşlarının
// notlarını/devamsızlığını/notlarını gösterir.
package scopeguard

import (
	"github.com/google/uuid"

	"lms/backend/pkg/core"
	"lms/backend/pkg/domain/users"
)

// StudentSubject, istenen öğrenci filtresini role göre sabitler: öğrenci için
// filtre her zaman kendisidir ve başkasının kimliği istenirse ErrForbidden döner.
// Diğer roller için istek olduğu gibi döner (kapsamı ayrı middleware uygular).
func StudentSubject(role users.Role, self uuid.UUID, requested *uuid.UUID) (*uuid.UUID, error) {
	if role != users.RoleStudent {
		return requested, nil
	}
	if requested != nil && *requested != self {
		return nil, core.ErrForbidden
	}
	id := self
	return &id, nil
}

// SharedNotesOnly: öğrenci yalnızca "öğrenci görebilir" işaretli notları görür;
// bu, istemcinin gönderdiği `shared` parametresine bırakılmaz.
func SharedNotesOnly(role users.Role, requested bool) bool {
	return requested || role == users.RoleStudent
}

// MayReadNote, tek bir notu okuma izni verir: öğrenci özel öğretmen notunu okuyamaz.
func MayReadNote(role users.Role, visibleToStudent bool) bool {
	return role != users.RoleStudent || visibleToStudent
}

// Viewer, liste uçlarında kimin baktığını taşır. Okul/kampüs/ders listeleri bu
// bilgiyle kullanıcının görebildikleriyle sınırlanır. nil Viewer sınırsızdır ve
// yalnızca iç çağrılar içindir.
type Viewer struct {
	UserID uuid.UUID
	Role   users.Role
}
