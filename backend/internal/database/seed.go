package database

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"lms/backend/pkg/domain/users"
)

// EnsureSuperAdmin, sistemde superadmin yoksa ilk yönetici hesabını oluşturur.
// Kayıt endpoint'i yalnızca öğrenci ürettiği için yönetici hesabının
// ortam değişkenlerinden bootstrap edilmesi gerekir.
func EnsureSuperAdmin(ctx context.Context, db *gorm.DB, email string, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return nil
	}
	var count int64
	if err := db.WithContext(ctx).Model(&users.User{}).
		Where("role = ?", users.RoleSuperAdmin).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	service := users.NewService(NewUserRepository(db))
	if _, err := service.Create(ctx, users.CreateInput{
		Name:     "System",
		Surname:  "Administrator",
		Email:    email,
		Password: password,
		Role:     users.RoleSuperAdmin,
	}); err != nil {
		return err
	}
	return nil
}
