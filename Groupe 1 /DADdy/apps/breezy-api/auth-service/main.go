package main

import (
	"log"

	"daddy/apps/breezy-api/internal/shared"
)

func main() {
	cfg := loadAuthConfig()

	db, err := shared.ConnectPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect auth-service database: %v", err)
	}

	if err := shared.AutoMigrate(db,
		&authUserModel{},
		&authRefreshTokenModel{},
		&authEmailVerificationModel{},
		&authPasswordResetModel{},
		&authMFAModel{},
		&authMFABackupCodeModel{},
		&authMFAPendingModel{},
		&authSocialAccountModel{},
		&authOAuthExchangeModel{},
	); err != nil {
		log.Fatalf("failed to migrate auth-service database: %v", err)
	}

	service := newAuthService(db, cfg)
	router := newAuthRouter(service)

	log.Printf("auth-service listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
