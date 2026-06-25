package main

import (
	"log"

	"daddy/apps/breezy-api/internal/shared"
)

func main() {
	cfg := loadUserConfig()

	db, err := shared.ConnectPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect user-service database: %v", err)
	}

	if err := shared.AutoMigrate(db, &userModel{}, &followModel{}, &followRequestModel{}, &sanctionModel{}, &blockModel{}); err != nil {
		log.Fatalf("failed to migrate user-service database: %v", err)
	}

	mc, err := newMinioClient(cfg.Minio)
	if err != nil {
		log.Fatalf("failed to connect user-service MinIO: %v", err)
	}

	service := newUserService(db, mc, cfg)
	router := newUserRouter(service)

	log.Printf("user-service listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
