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

	"daddy/apps/breezy-api/internal/shared"
)

var sharedDSN string

func TestMain(m *testing.M) {
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("daddy_msg_test"),
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
	if err := shared.AutoMigrate(db, &conversationModel{}, &participantModel{}, &messageModel{}); err != nil {
		panic("migrate: " + err.Error())
	}

	// Active le chiffrement au repos pour valider le round-trip de bout en bout.
	if err := initEncryption("test-message-encryption-key"); err != nil {
		panic("init encryption: " + err.Error())
	}

	code := m.Run()

	_ = ctr.Terminate(ctx)
	os.Exit(code)
}

// newTestApp ouvre une connexion neuve, vide les tables et renvoie l'app + son
// routeur. notifURL vide => notifications no-op (pas d'appel HTTP externe).
func newTestApp(t *testing.T) (*app, *gin.Engine) {
	t.Helper()
	db, err := shared.ConnectPostgres(sharedDSN)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	truncateAll(t, db)
	a := &app{db: db, internalKey: "test-internal-key"}
	return a, newRouter(a)
}

func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`TRUNCATE conversations, participants, messages RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// seedConversation crée une conversation entre deux utilisateurs et renvoie son ID.
func seedConversation(t *testing.T, a *app, userA, userB string) int64 {
	t.Helper()
	conv := conversationModel{}
	if err := a.db.Create(&conv).Error; err != nil {
		t.Fatalf("seed conversation: %v", err)
	}
	now := time.Now()
	parts := []participantModel{
		{ConversationID: conv.ID, UserID: userA, LastReadAt: now},
		{ConversationID: conv.ID, UserID: userB, LastReadAt: now},
	}
	if err := a.db.Create(&parts).Error; err != nil {
		t.Fatalf("seed participants: %v", err)
	}
	return conv.ID
}
