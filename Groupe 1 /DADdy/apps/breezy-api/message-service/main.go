package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"daddy/apps/breezy-api/internal/shared"
)

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	keyError          = "error"
	keyData           = "data"
	msgInvalidBody    = "invalid request body"
	msgFetchConv      = "failed to fetch conversation"
	msgFetchMessages  = "failed to fetch messages"
	msgFetchConvs     = "failed to fetch conversations"
	msgCreateConv     = "failed to create conversation"
	maxMessageContent = 2000
)

// ── Models ────────────────────────────────────────────────────────────────────

type conversationModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (conversationModel) TableName() string { return "conversations" }

type participantModel struct {
	ConversationID int64             `gorm:"primaryKey"`
	UserID         string            `gorm:"primaryKey;size:64"`
	LastReadAt     time.Time         `gorm:"not null;default:now()"`
	CreatedAt      time.Time         `gorm:"not null;default:now()"`
	Conversation   conversationModel `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE"`
}

func (participantModel) TableName() string { return "participants" }

type messageModel struct {
	ID             int64  `gorm:"primaryKey;autoIncrement"`
	ConversationID int64  `gorm:"not null;index:idx_msg_conv"`
	SenderID       string `gorm:"size:64;not null"`
	// Stocké chiffré (AES-GCM, cf. crypto.go) → type text car le ciphertext
	// base64 dépasse la limite de 2000 caractères du clair.
	Content   string    `gorm:"type:text;not null;default:''"`
	MediaURL  *string   `gorm:"size:500"`
	MediaType *string   `gorm:"size:10"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
}

func (messageModel) TableName() string { return "messages" }

// ── DTOs ──────────────────────────────────────────────────────────────────────

type msgResponse struct {
	ID             int64   `json:"id"`
	ConversationID int64   `json:"conversation_id"`
	SenderID       string  `json:"sender_id"`
	Content        string  `json:"content"`
	MediaURL       *string `json:"media_url,omitempty"`
	MediaType      *string `json:"media_type,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

type conversationResponse struct {
	ID           int64        `json:"id"`
	CreatedAt    string       `json:"created_at"`
	UpdatedAt    string       `json:"updated_at"`
	Participants []string     `json:"participants"`
	LastMessage  *msgResponse `json:"last_message"`
	UnreadCount  int64        `json:"unread_count"`
}

func toMsgResponse(m messageModel) msgResponse {
	return msgResponse{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Content:        decryptContent(m.Content),
		MediaURL:       m.MediaURL,
		MediaType:      m.MediaType,
		CreatedAt:      m.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func parseConvID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid conversation id")
		return 0, false
	}
	return id, true
}

func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return (page - 1) * limit, limit
}

// ── App ───────────────────────────────────────────────────────────────────────

type app struct {
	db          *gorm.DB
	notifURL    string
	internalKey string
}

func (a *app) sendNotifAsync(userID, actorID, actorUsername, convIDStr string) {
	if a.notifURL == "" {
		return
	}
	go func() {
		data, _ := json.Marshal(map[string]string{
			"user_id":        userID,
			"type":           "new_message",
			"actor_id":       actorID,
			"actor_username": actorUsername,
			"entity_id":      convIDStr,
			"entity_type":    "conversation",
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.notifURL+"/internal/notifications", bytes.NewReader(data))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Key", a.internalKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("sendNotifAsync: %v", err)
			return
		}
		if resp.StatusCode >= 300 {
			log.Printf("sendNotifAsync: unexpected status %s", resp.Status)
		}
		_ = resp.Body.Close()
	}()
}

func (a *app) callerID(c *gin.Context) string {
	id, _ := shared.IdentityFromContext(c)
	return id.UserID
}

// requireParticipant loads the conversation and verifies the caller is a member.
func (a *app) requireParticipant(c *gin.Context, convID int64, callerID string) (conversationModel, bool) {
	var conv conversationModel
	if err := a.db.First(&conv, convID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "conversation not found")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchConv})
		}
		return conversationModel{}, false
	}
	var count int64
	if err := a.db.Model(&participantModel{}).
		Where("conversation_id = ? AND user_id = ?", convID, callerID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to check conversation membership"})
		return conversationModel{}, false
	}
	if count == 0 {
		shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "not a participant in this conversation")
		return conversationModel{}, false
	}
	return conv, true
}

// buildConvResponse fetches participants, last message, and unread count for a single conversation.
func (a *app) buildConvResponse(conv conversationModel, callerID string) conversationResponse {
	var parts []participantModel
	a.db.Where("conversation_id = ?", conv.ID).Find(&parts)

	participants := make([]string, 0, len(parts))
	var lastReadAt time.Time
	for _, p := range parts {
		participants = append(participants, p.UserID)
		if p.UserID == callerID {
			lastReadAt = p.LastReadAt
		}
	}

	var lastMsg *msgResponse
	var msg messageModel
	if err := a.db.Where("conversation_id = ?", conv.ID).Order("created_at DESC").First(&msg).Error; err == nil {
		m := toMsgResponse(msg)
		lastMsg = &m
	}

	var unreadCount int64
	a.db.Model(&messageModel{}).
		Where("conversation_id = ? AND sender_id != ? AND created_at > ?", conv.ID, callerID, lastReadAt).
		Count(&unreadCount)

	return conversationResponse{
		ID:           conv.ID,
		CreatedAt:    conv.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    conv.UpdatedAt.UTC().Format(time.RFC3339),
		Participants: participants,
		LastMessage:  lastMsg,
		UnreadCount:  unreadCount,
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// POST /conversations — returns existing conversation or creates a new one.
func (a *app) getOrCreateConversation(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)

	var body struct {
		ParticipantID string `json:"participant_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, msgInvalidBody)
		return
	}
	otherID := strings.TrimSpace(body.ParticipantID)
	if otherID == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "participant_id is required")
		return
	}
	if otherID == identity.UserID {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "cannot start a conversation with yourself")
		return
	}

	// Look for an existing 1-on-1 conversation between the two users.
	var convID int64
	if err := a.db.Raw(`
		SELECT p1.conversation_id FROM participants p1
		JOIN participants p2 ON p2.conversation_id = p1.conversation_id AND p2.user_id = ?
		WHERE p1.user_id = ?
		LIMIT 1
	`, otherID, identity.UserID).Scan(&convID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchConv})
		return
	}

	if convID != 0 {
		var conv conversationModel
		if err := a.db.First(&conv, convID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchConv})
			return
		}
		c.JSON(http.StatusOK, gin.H{keyData: a.buildConvResponse(conv, identity.UserID)})
		return
	}

	// Create a new conversation with both participants atomically.
	tx := a.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgCreateConv})
		return
	}

	var conv conversationModel
	if err := tx.Create(&conv).Error; err != nil {
		_ = tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgCreateConv})
		return
	}
	now := time.Now()
	parts := []participantModel{
		{ConversationID: conv.ID, UserID: identity.UserID, LastReadAt: now},
		{ConversationID: conv.ID, UserID: otherID, LastReadAt: now},
	}
	if err := tx.Create(&parts).Error; err != nil {
		_ = tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgCreateConv})
		return
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgCreateConv})
		return
	}

	c.JSON(http.StatusCreated, gin.H{keyData: a.buildConvResponse(conv, identity.UserID)})
}

// GET /conversations — lists all conversations the caller participates in.
func (a *app) listConversations(c *gin.Context) {
	callerID := a.callerID(c)

	var myParts []participantModel
	if err := a.db.Where("user_id = ?", callerID).Find(&myParts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchConvs})
		return
	}
	if len(myParts) == 0 {
		c.JSON(http.StatusOK, gin.H{keyData: []conversationResponse{}})
		return
	}

	convIDs := make([]int64, len(myParts))
	for i, p := range myParts {
		convIDs[i] = p.ConversationID
	}

	var convs []conversationModel
	if err := a.db.Where("id IN ?", convIDs).Order("updated_at DESC").Find(&convs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchConvs})
		return
	}

	// Batch: all participants for each conversation.
	var allParts []participantModel
	if err := a.db.Where("conversation_id IN ?", convIDs).Find(&allParts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to fetch participants"})
		return
	}
	partMap := make(map[int64][]string)
	for _, p := range allParts {
		partMap[p.ConversationID] = append(partMap[p.ConversationID], p.UserID)
	}

	// Batch: last message per conversation (PostgreSQL DISTINCT ON).
	var lastMsgs []messageModel
	a.db.Raw(`
		SELECT DISTINCT ON (conversation_id) *
		FROM messages
		WHERE conversation_id IN ?
		ORDER BY conversation_id, created_at DESC
	`, convIDs).Scan(&lastMsgs)
	lastMsgMap := make(map[int64]msgResponse)
	for _, m := range lastMsgs {
		lastMsgMap[m.ConversationID] = toMsgResponse(m)
	}

	// Batch: unread counts per conversation.
	type unreadRow struct {
		ConversationID int64
		Count          int64
	}
	var unreadRows []unreadRow
	a.db.Raw(`
		SELECT m.conversation_id, COUNT(*) AS count
		FROM messages m
		JOIN participants p ON p.conversation_id = m.conversation_id AND p.user_id = ?
		WHERE m.conversation_id IN ?
		  AND m.sender_id != ?
		  AND m.created_at > p.last_read_at
		GROUP BY m.conversation_id
	`, callerID, convIDs, callerID).Scan(&unreadRows)
	unreadMap := make(map[int64]int64)
	for _, u := range unreadRows {
		unreadMap[u.ConversationID] = u.Count
	}

	out := make([]conversationResponse, 0, len(convs))
	for _, conv := range convs {
		var lm *msgResponse
		if m, ok := lastMsgMap[conv.ID]; ok {
			lm = &m
		}
		out = append(out, conversationResponse{
			ID:           conv.ID,
			CreatedAt:    conv.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:    conv.UpdatedAt.UTC().Format(time.RFC3339),
			Participants: partMap[conv.ID],
			LastMessage:  lm,
			UnreadCount:  unreadMap[conv.ID],
		})
	}
	c.JSON(http.StatusOK, gin.H{keyData: out})
}

// GET /conversations/:id.
func (a *app) getConversation(c *gin.Context) {
	callerID := a.callerID(c)
	convID, ok := parseConvID(c)
	if !ok {
		return
	}
	conv, ok := a.requireParticipant(c, convID, callerID)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{keyData: a.buildConvResponse(conv, callerID)})
}

// GET /conversations/:id/messages.
func (a *app) listMessages(c *gin.Context) {
	callerID := a.callerID(c)
	convID, ok := parseConvID(c)
	if !ok {
		return
	}
	if _, ok := a.requireParticipant(c, convID, callerID); !ok {
		return
	}

	offset, limit := parsePagination(c)
	var msgs []messageModel
	if err := a.db.Where("conversation_id = ?", convID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&msgs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchMessages})
		return
	}

	out := make([]msgResponse, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, toMsgResponse(m))
	}
	c.JSON(http.StatusOK, gin.H{keyData: out})
}

// POST /conversations/:id/messages.
func (a *app) sendMessage(c *gin.Context) {
	identity := c.MustGet("identity").(shared.Identity)
	convID, ok := parseConvID(c)
	if !ok {
		return
	}
	if _, ok := a.requireParticipant(c, convID, identity.UserID); !ok {
		return
	}

	var body struct {
		Content   string  `json:"content"`
		MediaURL  *string `json:"media_url"`
		MediaType *string `json:"media_type"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, msgInvalidBody)
		return
	}
	content := strings.TrimSpace(body.Content)
	if len(content) > maxMessageContent {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "content exceeds 2000 characters")
		return
	}
	if content == "" && body.MediaURL == nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "content or media is required")
		return
	}
	if body.MediaURL != nil {
		mt := ""
		if body.MediaType != nil {
			mt = strings.TrimSpace(*body.MediaType)
		}
		if mt != "image" && mt != "video" && mt != "audio" {
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "media_type must be 'image', 'video', or 'audio'")
			return
		}
		body.MediaType = &mt
	}

	// Le contenu est chiffré avant insertion ; toMsgResponse le déchiffre pour
	// la réponse, l'appelant reçoit donc bien le clair.
	msg := messageModel{ConversationID: convID, SenderID: identity.UserID, Content: encryptContent(content), MediaURL: body.MediaURL, MediaType: body.MediaType}
	if err := a.db.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{keyError: "failed to send message"})
		return
	}
	sentAt := msg.CreatedAt
	a.db.Model(&conversationModel{}).Where("id = ?", convID).Update("updated_at", sentAt)
	a.db.Model(&participantModel{}).
		Where("conversation_id = ? AND user_id = ?", convID, identity.UserID).
		Update("last_read_at", sentAt)

	var otherParts []participantModel
	a.db.Where("conversation_id = ? AND user_id != ?", convID, identity.UserID).Find(&otherParts)
	convIDStr := strconv.FormatInt(convID, 10)
	for _, p := range otherParts {
		a.sendNotifAsync(p.UserID, identity.UserID, identity.Username, convIDStr)
	}

	c.JSON(http.StatusCreated, gin.H{keyData: toMsgResponse(msg)})
}

// PUT /conversations/:id/read.
func (a *app) markRead(c *gin.Context) {
	callerID := a.callerID(c)
	convID, ok := parseConvID(c)
	if !ok {
		return
	}
	if _, ok := a.requireParticipant(c, convID, callerID); !ok {
		return
	}
	a.db.Model(&participantModel{}).
		Where("conversation_id = ? AND user_id = ?", convID, callerID).
		Update("last_read_at", time.Now())
	c.Status(http.StatusNoContent)
}

// newRouter assemble les routes du message-service. Extrait de main pour être
// instanciable depuis les tests (cf. newUserRouter / newAuthRouter).
func newRouter(a *app) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.HandleMethodNotAllowed = true

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, shared.NewHealthResponse("message-service"))
	})

	auth := shared.RequireIdentity()
	router.POST("/conversations", auth, a.getOrCreateConversation)
	router.GET("/conversations", auth, a.listConversations)
	router.GET("/conversations/:id", auth, a.getConversation)
	router.GET("/conversations/:id/messages", auth, a.listMessages)
	router.POST("/conversations/:id/messages", auth, a.sendMessage)
	router.PUT("/conversations/:id/read", auth, a.markRead)

	router.NoMethod(func(c *gin.Context) {
		c.String(http.StatusMethodNotAllowed, "method not allowed")
	})

	return router
}
