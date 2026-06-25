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
		postgres.WithDatabase("daddy_post_test"),
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
		&postModel{}, &likeModel{}, &rebreezersModel{}, &mediaModel{}, &hashtagModel{},
		&postHashtagModel{}, &postMentionModel{}, &pollModel{}, &pollOptionModel{},
		&pollVoteModel{}, &reportModel{}, &bookmarkModel{}, &userAffinityModel{}, &hashtagNeighborModel{}, &hashtagClusterModel{},
	); err != nil {
		panic("migrate: " + err.Error())
	}

	code := m.Run()

	_ = ctr.Terminate(ctx)
	os.Exit(code)
}

// newTestApp ouvre une connexion neuve, vide les tables, et renvoie l'app + son
// routeur. notifURL/userURL vides => aucun appel HTTP externe (notif no-op,
// canViewProfile autorise). mc nil : les tests ne touchent pas de média.
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
	if err := db.Exec(`TRUNCATE posts, likes, rebreezers, media, hashtags, post_hashtags, post_mentions, polls, poll_options, poll_votes, reports, bookmarks, user_affinity, hashtag_neighbors, hashtag_clusters RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// seedPost insère un post et renvoie son ID.
func seedPost(t *testing.T, a *app, authorID, content string) int64 {
	t.Helper()
	p := postModel{Content: content, AuthorID: authorID, AuthorUsername: authorID}
	if err := a.db.Create(&p).Error; err != nil {
		t.Fatalf("seed post: %v", err)
	}
	return p.ID
}

// reloadPost relit un post par ID.
func reloadPost(t *testing.T, a *app, id int64) postModel {
	t.Helper()
	var p postModel
	if err := a.db.First(&p, id).Error; err != nil {
		t.Fatalf("reload post %d: %v", id, err)
	}
	return p
}
