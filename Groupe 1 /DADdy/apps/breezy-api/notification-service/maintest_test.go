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

func init() { gin.SetMode(gin.TestMode) }

func TestMain(m *testing.M) {
	ctx := context.Background()

	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("daddy_notif_test"),
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
	if err := shared.AutoMigrate(db, &notificationModel{}, &subscriptionModel{}); err != nil {
		panic("migrate: " + err.Error())
	}

	code := m.Run()

	_ = ctr.Terminate(ctx)
	os.Exit(code)
}

const testInternalKey = "test-internal-key"

// newTestApp ouvre une connexion neuve, vide les tables, et renvoie l'app + son
// routeur prêts à l'emploi. PostgresDSN est renseigné pour les handlers qui en
// dépendent (le SSE n'est pas testé ici).
func newTestApp(t *testing.T) (*app, *gin.Engine) {
	t.Helper()
	db, err := shared.ConnectPostgres(sharedDSN)
	if err != nil {
		t.Fatalf("connect test db: %v", err)
	}
	truncateAll(t, db)
	a := &app{db: db, cfg: notifConfig{InternalKey: testInternalKey, PostgresDSN: sharedDSN}}
	return a, newRouter(a)
}

func truncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`TRUNCATE notifications, subscriptions RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// seedNotif insère une notification pour userID et renvoie son ID.
func seedNotif(t *testing.T, a *app, userID, actorID string) int64 {
	t.Helper()
	n := notificationModel{UserID: userID, Type: "follow", ActorID: actorID, ActorUsername: "actor", EntityType: "user"}
	if err := a.db.Create(&n).Error; err != nil {
		t.Fatalf("seed notif: %v", err)
	}
	return n.ID
}
