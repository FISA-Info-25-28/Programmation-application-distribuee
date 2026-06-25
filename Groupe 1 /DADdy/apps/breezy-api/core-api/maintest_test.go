//go:build integration

package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"

	"daddy/apps/breezy-api/internal/models"
	"daddy/apps/breezy-api/internal/shared"
)

var sharedDSN string

func TestMain(m *testing.M) {
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("daddy_core_test"),
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
	if err := shared.AutoMigrate(db,
		&models.User{}, &models.Post{}, &models.PostTag{},
		&models.Comment{}, &models.Like{},
	); err != nil {
		panic("migrate: " + err.Error())
	}

	code := m.Run()

	_ = ctr.Terminate(ctx)
	os.Exit(code)
}

// newTestServer ouvre une connexion neuve, vide les tables, et renvoie le
// serveur + son routeur.
func newTestServer(t *testing.T) (*server, *gin.Engine) {
	t.Helper()
	db, err := shared.ConnectPostgres(sharedDSN)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	truncateAll(t, db)
	s := &server{db: db, internalSecret: "test-internal-key"}
	return s, newRouter(s)
}

func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`TRUNCATE users, posts, post_tags, comments, likes RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// seedUser insère un utilisateur minimal (les FK des posts/comments l'exigent).
func seedUser(t *testing.T, s *server, id, username string) {
	t.Helper()
	u := models.User{
		ID: id, Username: username, Email: id + "@example.com",
		Password: "-", Role: "user", Status: "active", Language: "fr", Theme: "system",
	}
	if err := s.db.Create(&u).Error; err != nil {
		t.Fatalf("seed user %s: %v", id, err)
	}
}

// seedPost insère un post et renvoie son ID.
func seedPost(t *testing.T, s *server, authorID, content string) int64 {
	t.Helper()
	p := models.Post{Content: content, AuthorID: authorID}
	if err := s.db.Create(&p).Error; err != nil {
		t.Fatalf("seed post: %v", err)
	}
	return p.ID
}

func reloadPost(t *testing.T, s *server, id int64) models.Post {
	t.Helper()
	var p models.Post
	if err := s.db.First(&p, id).Error; err != nil {
		t.Fatalf("reload post %d: %v", id, err)
	}
	return p
}
