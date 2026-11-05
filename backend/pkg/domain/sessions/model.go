package sessions

import (
	"time"

	"github.com/google/uuid"

	"lms/backend/pkg/domain/base"
)

type Session struct {
	base.Model
	UserID    uuid.UUID  `gorm:"type:char(36);not null;index" json:"user_id"`
	UserAgent string     `gorm:"size:512" json:"user_agent"`
	JWTKey    string     `gorm:"size:255;not null" json:"jwt_key"`
	ExpiresAt time.Time  `gorm:"index;not null" json:"expires_at"`
	RevokedAt *time.Time `gorm:"index" json:"revoked_at"`
}
