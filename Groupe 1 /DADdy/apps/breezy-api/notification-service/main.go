package main

import (
	"fmt"
	"hash/fnv"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"daddy/apps/breezy-api/internal/shared"
)

const (
	keyError             = "error"
	keyData              = "data"
	keyCount             = "count"
	msgInvalidBody       = "invalid body"
	msgStreamUnavailable = "stream unavailable"
)

// ── Models ────────────────────────────────────────────────────────────────────

type notificationModel struct {
	ID            int64      `gorm:"primaryKey;autoIncrement"`
	UserID        string     `gorm:"size:64;not null;index:idx_notif_user;index:idx_notif_user_read,priority:1"`
	Type          string     `gorm:"size:20;not null"` // like, comment, follow, new_post.
	ActorID       string     `gorm:"size:64;not null"`
	ActorUsername string     `gorm:"size:50"`
	EntityID      string     `gorm:"size:64"`
	EntityType    string     `gorm:"size:20"` // post, user.
	ReadAt        *time.Time `gorm:"index:idx_notif_user_read,priority:2"`
	CreatedAt     time.Time  `gorm:"not null;autoCreateTime"`
}

func (notificationModel) TableName() string { return "notifications" }

type subscriptionModel struct {
	SubscriberID string    `gorm:"primaryKey;size:64"`
	TargetUserID string    `gorm:"primaryKey;size:64;index:idx_sub_target"`
	CreatedAt    time.Time `gorm:"not null;autoCreateTime"`
}

func (subscriptionModel) TableName() string { return "subscriptions" }

// ── DTOs ──────────────────────────────────────────────────────────────────────

type notifResponse struct {
	ID            int64  `json:"id"`
	Type          string `json:"type"`
	ActorID       string `json:"actor_id"`
	ActorUsername string `json:"actor_username"`
	EntityID      string `json:"entity_id"`
	EntityType    string `json:"entity_type"`
	Read          bool   `json:"read"`
	CreatedAt     string `json:"created_at"`
}

func toNotifResponse(n notificationModel) notifResponse {
	return notifResponse{
		ID:            n.ID,
		Type:          n.Type,
		ActorID:       n.ActorID,
		ActorUsername: n.ActorUsername,
		EntityID:      n.EntityID,
		EntityType:    n.EntityType,
		Read:          n.ReadAt != nil,
		CreatedAt:     n.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// ── Config ────────────────────────────────────────────────────────────────────

type notifConfig struct {
	Port        string
	PostgresDSN string
	InternalKey string
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// userChannel derives a safe 17-char PostgreSQL identifier from a user ID.
// Uses fnv64a (non-cryptographic) to stay within the 63-char NAMEDATALEN limit.
func userChannel(userID string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(userID))
	return fmt.Sprintf("n%016x", h.Sum64())
}

// ── App ───────────────────────────────────────────────────────────────────────

type app struct {
	db  *gorm.DB
	cfg notifConfig
}

// notifyUser fires a pg_notify for the given user's channel (best-effort).
func (a *app) notifyUser(userID string) {
	a.db.Exec("SELECT pg_notify($1, '')", userChannel(userID))
}

// ── Notification handlers ─────────────────────────────────────────────────────

func (a *app) listNotifications(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var total int64
	a.db.Model(&notificationModel{}).Where("user_id = ?", identity.UserID).Count(&total)

	var notifs []notificationModel
	if err := a.db.Where("user_id = ?", identity.UserID).
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&notifs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to fetch notifications"})
		return
	}

	out := make([]notifResponse, len(notifs))
	for i, n := range notifs {
		out[i] = toNotifResponse(n)
	}
	c.JSON(http.StatusOK, gin.H{keyData: out, "total": total, "page": page, "limit": limit})
}

func (a *app) unreadCount(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	var count int64
	a.db.Model(&notificationModel{}).Where("user_id = ? AND read_at IS NULL", identity.UserID).Count(&count)
	c.JSON(http.StatusOK, gin.H{keyData: gin.H{keyCount: count}})
}

func (a *app) markAllRead(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	now := time.Now().UTC()
	a.db.Model(&notificationModel{}).
		Where("user_id = ? AND read_at IS NULL", identity.UserID).
		Update("read_at", now)
	c.Status(http.StatusNoContent)
}

func (a *app) markRead(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	notifID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid notification id")
		return
	}
	now := time.Now().UTC()
	res := a.db.Model(&notificationModel{}).
		Where("id = ? AND user_id = ?", notifID, identity.UserID).
		Update("read_at", now)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to mark as read"})
		return
	}
	if res.RowsAffected == 0 {
		shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "notification not found")
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *app) deleteNotification(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	notifID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid notification id")
		return
	}
	res := a.db.Where("id = ? AND user_id = ?", notifID, identity.UserID).Delete(&notificationModel{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to delete notification"})
		return
	}
	if res.RowsAffected == 0 {
		shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "notification not found")
		return
	}
	c.Status(http.StatusNoContent)
}

// ── Subscription handlers ─────────────────────────────────────────────────────

func (a *app) subscribe(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	var body struct {
		TargetUserID string `json:"target_user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "target_user_id required")
		return
	}
	if body.TargetUserID == identity.UserID {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "cannot subscribe to yourself")
		return
	}
	res := a.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&subscriptionModel{
		SubscriberID: identity.UserID,
		TargetUserID: body.TargetUserID,
	})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to subscribe"})
		return
	}
	c.Status(http.StatusCreated)
}

func (a *app) unsubscribe(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	targetID := c.Param("userId")
	a.db.Where("subscriber_id = ? AND target_user_id = ?", identity.UserID, targetID).Delete(&subscriptionModel{})
	c.Status(http.StatusNoContent)
}

func (a *app) listSubscriptions(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	var subs []subscriptionModel
	a.db.Where("subscriber_id = ?", identity.UserID).Find(&subs)
	ids := make([]string, len(subs))
	for i, s := range subs {
		ids[i] = s.TargetUserID
	}
	c.JSON(http.StatusOK, gin.H{keyData: ids})
}

// ── Internal subscription handlers ───────────────────────────────────────────

func (a *app) internalSubscribe(c *gin.Context) {
	var req struct {
		SubscriberID string `json:"subscriber_id" binding:"required"`
		TargetUserID string `json:"target_user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{keyError: msgInvalidBody})
		return
	}
	if req.SubscriberID == req.TargetUserID {
		c.Status(http.StatusNoContent)
		return
	}
	a.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&subscriptionModel{
		SubscriberID: req.SubscriberID,
		TargetUserID: req.TargetUserID,
	})
	c.Status(http.StatusNoContent)
}

func (a *app) internalUnsubscribe(c *gin.Context) {
	var req struct {
		SubscriberID string `json:"subscriber_id" binding:"required"`
		TargetUserID string `json:"target_user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{keyError: msgInvalidBody})
		return
	}
	a.db.Where("subscriber_id = ? AND target_user_id = ?", req.SubscriberID, req.TargetUserID).Delete(&subscriptionModel{})
	c.Status(http.StatusNoContent)
}

// ── Internal middleware + handlers ────────────────────────────────────────────

func (a *app) requireInternalKey(c *gin.Context) {
	if c.GetHeader("X-Internal-Key") != a.cfg.InternalKey {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{keyError: "unauthorized"})
		return
	}
	c.Next()
}

func (a *app) internalCreateNotif(c *gin.Context) {
	var req struct {
		UserID        string `json:"user_id" binding:"required"`
		Type          string `json:"type" binding:"required"`
		ActorID       string `json:"actor_id" binding:"required"`
		ActorUsername string `json:"actor_username"`
		EntityID      string `json:"entity_id"`
		EntityType    string `json:"entity_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{keyError: msgInvalidBody})
		return
	}
	if req.ActorID == req.UserID {
		c.Status(http.StatusNoContent)
		return
	}
	n := notificationModel{
		UserID:        req.UserID,
		Type:          req.Type,
		ActorID:       req.ActorID,
		ActorUsername: req.ActorUsername,
		EntityID:      req.EntityID,
		EntityType:    req.EntityType,
	}
	if err := a.db.Create(&n).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to create notification"})
		return
	}
	a.notifyUser(req.UserID)
	c.Status(http.StatusCreated)
}

func (a *app) internalNewPost(c *gin.Context) {
	var req struct {
		AuthorID       string `json:"author_id" binding:"required"`
		AuthorUsername string `json:"author_username"`
		PostID         string `json:"post_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{keyError: msgInvalidBody})
		return
	}

	var subs []subscriptionModel
	a.db.Where("target_user_id = ? AND subscriber_id != ?", req.AuthorID, req.AuthorID).Find(&subs)
	if len(subs) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	notifs := make([]notificationModel, 0, len(subs))
	for _, s := range subs {
		notifs = append(notifs, notificationModel{
			UserID:        s.SubscriberID,
			Type:          "new_post",
			ActorID:       req.AuthorID,
			ActorUsername: req.AuthorUsername,
			EntityID:      req.PostID,
			EntityType:    "post",
		})
	}
	if err := a.db.Create(&notifs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to create notifications"})
		return
	}
	for _, s := range subs {
		a.notifyUser(s.SubscriberID)
	}
	c.Status(http.StatusCreated)
}

// ── Router ──────────────────────────────────────────────────────────────────

// newRouter assemble les routes du notif-service. Extrait de main pour être
// instanciable depuis les tests (cf. newUserRouter / newAuthRouter).
func newRouter(a *app) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.HandleMethodNotAllowed = true

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, shared.NewHealthResponse("notif-service"))
	})

	auth := shared.RequireIdentity()
	router.GET("/notifications/stream", auth, a.streamNotifications)
	router.GET("/notifications", auth, a.listNotifications)
	router.GET("/notifications/unread-count", auth, a.unreadCount)
	router.PUT("/notifications/read-all", auth, a.markAllRead)
	router.PUT("/notifications/:id/read", auth, a.markRead)
	router.DELETE("/notifications/:id", auth, a.deleteNotification)

	router.POST("/subscriptions", auth, a.subscribe)
	router.DELETE("/subscriptions/:userId", auth, a.unsubscribe)
	router.GET("/subscriptions", auth, a.listSubscriptions)

	internal := router.Group("/internal", a.requireInternalKey)
	internal.POST("/notifications", a.internalCreateNotif)
	internal.POST("/notifications/new-post", a.internalNewPost)
	internal.POST("/subscriptions", a.internalSubscribe)
	internal.DELETE("/subscriptions", a.internalUnsubscribe)

	router.NoMethod(func(c *gin.Context) {
		c.String(http.StatusMethodNotAllowed, "method not allowed")
	})

	return router
}
