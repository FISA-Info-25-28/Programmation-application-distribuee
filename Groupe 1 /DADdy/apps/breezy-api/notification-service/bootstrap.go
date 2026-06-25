package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"daddy/apps/breezy-api/internal/shared"
)

// Ce fichier regroupe le code d'infrastructure non couvrable par les tests :
// point d'entrée (main), chargement de la config et flux SSE temps réel
// (streamNotifications, qui ouvre une connexion pgx LISTEN et diffuse en continu).
// Isolé ici pour être exclu du calcul de couverture (voir .coverignore). La
// logique HTTP/métier testable reste dans main.go.

func loadNotifConfig() notifConfig {
	return notifConfig{
		Port:        shared.GetEnv("PORT", "3106"),
		PostgresDSN: shared.GetEnvAny([]string{"NOTIF_DATABASE_URL", "DATABASE_URL"}, "postgres://postgres:postgres@localhost:5432/daddy?sslmode=disable"),
		InternalKey: shared.SecretEnv("INTERNAL_API_KEY", "dev-internal-key"),
	}
}

func (a *app) streamNotifications(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)

	// Use a background context with timeout for setup so a racing client-disconnect
	// (React strict-mode double-mount, SSE reconnect) doesn't abort the pgx handshake
	// and produce a spurious 500.
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer setupCancel()

	conn, err := pgx.Connect(setupCtx, a.cfg.PostgresDSN)
	if err != nil {
		log.Printf("streamNotifications: pgx.Connect failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgStreamUnavailable})
		return
	}
	defer func() { _ = conn.Close(context.Background()) }()

	if _, err := conn.Exec(setupCtx, "LISTEN "+userChannel(identity.UserID)); err != nil {
		log.Printf("streamNotifications: LISTEN failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgStreamUnavailable})
		return
	}
	setupCancel() // setup done, release the timeout

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Flush()

	clientCtx := c.Request.Context()
	if clientCtx.Err() != nil {
		return // client already gone during setup, clean up silently
	}
	notifCh := make(chan struct{}, 1)

	go func() {
		for {
			if _, err := conn.WaitForNotification(clientCtx); err != nil {
				return
			}
			select {
			case notifCh <- struct{}{}:
			default:
			}
		}
	}()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-clientCtx.Done():
			return
		case <-heartbeat.C:
			_, _ = fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()
		case <-notifCh:
			var latest notificationModel
			a.db.Where("user_id = ? AND read_at IS NULL", identity.UserID).
				Order("created_at DESC").
				First(&latest)
			var count int64
			a.db.Model(&notificationModel{}).
				Where("user_id = ? AND read_at IS NULL", identity.UserID).
				Count(&count)
			payload, _ := json.Marshal(map[string]any{
				"count":          count,
				"type":           latest.Type,
				"actor_id":       latest.ActorID,
				"actor_username": latest.ActorUsername,
				"entity_id":      latest.EntityID,
			})
			_, _ = fmt.Fprintf(c.Writer, "event: new_notification\ndata: %s\n\n", payload)
			c.Writer.Flush()
		}
	}
}

func main() {
	cfg := loadNotifConfig()

	db, err := shared.ConnectPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect notif-service database: %v", err)
	}
	if err := shared.AutoMigrate(db, &notificationModel{}, &subscriptionModel{}); err != nil {
		log.Fatalf("failed to migrate notif-service database: %v", err)
	}

	a := &app{db: db, cfg: cfg}
	router := newRouter(a)

	log.Printf("notif-service listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
