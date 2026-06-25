//go:build integration

package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"

	"daddy/apps/breezy-api/internal/shared"
)

// pgContainer est démarré une seule fois pour tout le paquet (un conteneur Postgres
// est coûteux). Chaque test obtient ensuite une base propre via truncateAll.
var sharedDSN string

// TestMain démarre un Postgres jetable via testcontainers, applique les migrations
// GORM (les mêmes que main.go) puis lance la suite. Le conteneur est arrêté à la fin.
func TestMain(m *testing.M) {
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("daddy_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		panic("start postgres container: " + err.Error())
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("connection string: " + err.Error())
	}
	sharedDSN = dsn

	db, err := shared.ConnectPostgres(dsn)
	if err != nil {
		panic("connect postgres: " + err.Error())
	}
	if err := shared.AutoMigrate(db, &userModel{}, &followModel{}, &followRequestModel{}, &sanctionModel{}, &blockModel{}); err != nil {
		panic("migrate: " + err.Error())
	}

	code := m.Run()

	_ = ctr.Terminate(ctx)
	os.Exit(code)
}

// newTestService ouvre une connexion neuve sur la base partagée, la vide, et
// renvoie un userService prêt à l'emploi. mc est nil : aucun test d'intégration
// ici ne touche MinIO. NotifURL/AuthURL restent vides → pas d'appel externe.
func newTestService(t *testing.T) *userService {
	t.Helper()

	db, err := shared.ConnectPostgres(sharedDSN)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	truncateAll(t, db)

	return newUserService(db, nil, userConfig{})
}

// truncateAll vide toutes les tables entre deux tests pour les isoler.
func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`TRUNCATE users, follows, follow_requests, sanctions, blocks RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// seedUser insère un profil minimal et renvoie son ID.
func seedUser(t *testing.T, s *userService, id, username string, private bool) string {
	t.Helper()
	if err := s.db.Create(&userModel{
		ID:        id,
		Username:  username,
		Email:     username + "@example.com",
		Role:      roleUser,
		Status:    statusActive,
		IsPrivate: private,
	}).Error; err != nil {
		t.Fatalf("seed user %s: %v", id, err)
	}
	return id
}

// reload relit un userModel par ID.
func reload(t *testing.T, s *userService, id string) userModel {
	t.Helper()
	var u userModel
	if err := s.db.First(&u, "id = ?", id).Error; err != nil {
		t.Fatalf("reload %s: %v", id, err)
	}
	return u
}
