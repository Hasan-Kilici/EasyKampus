package main

import (
	"context"
	"log"

	"lms/backend/internal/app"
	"lms/backend/internal/config"
	"lms/backend/internal/database"
)

// @title						LMS Backend API
// @version					1.0
// @description				LMS and student/teacher performance API.
// @host						localhost:3000
// @BasePath					/api
// @schemes					http https
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
func main() {
	appConfig, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	application, db, err := app.New(appConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := database.Close(context.Background(), db); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	log.Fatal(application.Listen(":" + appConfig.Port))
}
