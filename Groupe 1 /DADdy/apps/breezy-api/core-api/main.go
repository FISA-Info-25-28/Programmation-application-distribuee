package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"daddy/apps/breezy-api/internal/models"
	"daddy/apps/breezy-api/internal/shared"
)

// ─── Constants ────────────────────────────────────────────────────────────────

const (
	keyError = "error"
	keyData  = "data"
	keyPage  = "page"
	keyLimit = "limit"
	keyTotal = "total"
)

// ─── DTOs ────────────────────────────────────────────────────────────────────

type userSummary struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	AvatarUrl *string `json:"avatarUrl"`
	Bio       *string `json:"bio"`
}

type postResponse struct {
	ID            string      `json:"id"`
	AuthorID      string      `json:"authorId"`
	Author        userSummary `json:"author"`
	Content       string      `json:"content"`
	Tags          []string    `json:"tags"`
	ParentID      *string     `json:"parentId"`
	LikesCount    int         `json:"likesCount"`
	CommentsCount int         `json:"commentsCount"`
	LikedByMe     bool        `json:"likedByMe"`
	CreatedAt     string      `json:"createdAt"`
	UpdatedAt     string      `json:"updatedAt"`
}

type commentResponse struct {
	ID              string      `json:"id"`
	PostID          string      `json:"postId"`
	ParentCommentID *string     `json:"parentCommentId"`
	AuthorID        string      `json:"authorId"`
	Author          userSummary `json:"author"`
	Content         string      `json:"content"`
	CreatedAt       string      `json:"createdAt"`
}

type createPostRequest struct {
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

type createCommentRequest struct {
	Content         string  `json:"content"`
	ParentCommentID *string `json:"parentCommentId"`
}

// ─── Server ───────────────────────────────────────────────────────────────────

type server struct {
	db             *gorm.DB
	internalSecret string
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func pageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}

func toPostID(id int64) string { return fmt.Sprintf("%d", id) }

func postTags(db *gorm.DB, postID int64) []string {
	var tags []models.PostTag
	db.Where("post_id = ?", postID).Find(&tags)
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		result = append(result, t.Tag)
	}
	return result
}

func hasLiked(db *gorm.DB, userID string, postID int64) bool {
	if userID == "" {
		return false
	}
	var count int64
	db.Model(&models.Like{}).Where("user_id = ? AND post_id = ?", userID, postID).Count(&count)
	return count > 0
}

func buildPostResponse(db *gorm.DB, post models.Post, callerID string) postResponse {
	var parentIDStr *string
	if post.ParentID != nil {
		s := toPostID(*post.ParentID)
		parentIDStr = &s
	}
	return postResponse{
		ID:       toPostID(post.ID),
		AuthorID: post.AuthorID,
		Author: userSummary{
			ID:        post.Author.ID,
			Username:  post.Author.Username,
			AvatarUrl: post.Author.ProfilePicture,
			Bio:       post.Author.Bio,
		},
		Content:       post.Content,
		Tags:          postTags(db, post.ID),
		ParentID:      parentIDStr,
		LikesCount:    post.LikesCount,
		CommentsCount: post.CommentsCount,
		LikedByMe:     hasLiked(db, callerID, post.ID),
		CreatedAt:     post.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     post.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func upsertTags(db *gorm.DB, postID int64, tags []string) {
	db.Where("post_id = ?", postID).Delete(&models.PostTag{})
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			continue
		}
		db.FirstOrCreate(&models.PostTag{PostID: postID, Tag: tag})
	}
}

// ─── Handlers ────────────────────────────────────────────────────────────────

func (s *server) health(c *gin.Context) {
	c.JSON(http.StatusOK, shared.NewHealthResponse("core-api"))
}

func (s *server) internalUsers(c *gin.Context) {
	if c.GetHeader("X-Internal-Key") != s.internalSecret {
		c.JSON(http.StatusForbidden, gin.H{keyError: "forbidden"})
		return
	}
	var req struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{keyError: "invalid body"})
		return
	}
	role := req.Role
	if role == "" {
		role = "user"
	}
	user := models.User{
		ID:       req.ID,
		Username: req.Username,
		Email:    req.Email,
		Password: "-",
		Role:     role,
		Status:   "active",
		Language: "fr",
		Theme:    "system",
	}
	s.db.FirstOrCreate(&user, "id = ?", req.ID)
	c.JSON(http.StatusCreated, gin.H{"id": user.ID})
}

func (s *server) getFeed(c *gin.Context) {
	page, limit := pageParams(c)
	offset := (page - 1) * limit
	caller, _ := shared.IdentityFromContext(c)

	var posts []models.Post
	var total int64
	s.db.Model(&models.Post{}).Where("parent_id IS NULL").Count(&total)
	if err := s.db.Preload("Author").
		Where("parent_id IS NULL").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&posts).Error; err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch feed")
		return
	}

	result := make([]postResponse, 0, len(posts))
	for _, p := range posts {
		result = append(result, buildPostResponse(s.db, p, caller.UserID))
	}
	c.JSON(http.StatusOK, gin.H{keyData: result, keyPage: page, keyLimit: limit, keyTotal: total})
}

func (s *server) createPost(c *gin.Context) {
	identity, ok := shared.IdentityFromContext(c)
	if !ok {
		shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "authentication required")
		return
	}

	var req createPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid body")
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "content is required")
		return
	}
	if len(req.Content) > 280 {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "content exceeds 280 characters")
		return
	}

	var author models.User
	if err := s.db.First(&author, "id = ?", identity.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusUnprocessableEntity, shared.ErrNotFound, "user profile not found; complete registration first")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to validate author")
		return
	}

	post := models.Post{
		Content:  req.Content,
		AuthorID: identity.UserID,
	}
	if err := s.db.Create(&post).Error; err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to create post")
		return
	}
	if len(req.Tags) > 0 {
		upsertTags(s.db, post.ID, req.Tags)
	}
	post.Author = author
	c.JSON(http.StatusCreated, buildPostResponse(s.db, post, identity.UserID))
}

func (s *server) getPost(c *gin.Context) {
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid post id")
		return
	}
	caller, _ := shared.IdentityFromContext(c)

	var post models.Post
	if err := s.db.Preload("Author").First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "post not found")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch post")
		return
	}
	c.JSON(http.StatusOK, buildPostResponse(s.db, post, caller.UserID))
}

func (s *server) deletePost(c *gin.Context) {
	identity, ok := shared.IdentityFromContext(c)
	if !ok {
		shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "authentication required")
		return
	}
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid post id")
		return
	}

	var post models.Post
	if err := s.db.First(&post, postID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "post not found")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch post")
		return
	}
	if post.AuthorID != identity.UserID && identity.Role != "moderator" && identity.Role != "administrator" {
		shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "not allowed")
		return
	}
	if err := s.db.Delete(&models.Post{}, postID).Error; err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to delete post")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *server) likePost(c *gin.Context) {
	identity, ok := shared.IdentityFromContext(c)
	if !ok {
		shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "authentication required")
		return
	}
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid post id")
		return
	}

	like := models.Like{UserID: identity.UserID, PostID: postID}
	result := s.db.Where(like).FirstOrCreate(&like)
	if result.Error != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to like post")
		return
	}
	if result.RowsAffected > 0 {
		s.db.Model(&models.Post{}).Where("id = ?", postID).UpdateColumn("likes_count", gorm.Expr("likes_count + 1"))
	}
	c.Status(http.StatusNoContent)
}

func (s *server) unlikePost(c *gin.Context) {
	identity, ok := shared.IdentityFromContext(c)
	if !ok {
		shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "authentication required")
		return
	}
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid post id")
		return
	}

	result := s.db.Where("user_id = ? AND post_id = ?", identity.UserID, postID).Delete(&models.Like{})
	if result.Error != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to unlike post")
		return
	}
	if result.RowsAffected > 0 {
		s.db.Model(&models.Post{}).Where("id = ?", postID).UpdateColumn("likes_count", gorm.Expr("GREATEST(likes_count - 1, 0)"))
	}
	c.Status(http.StatusNoContent)
}

func (s *server) getComments(c *gin.Context) {
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid post id")
		return
	}
	page, limit := pageParams(c)
	offset := (page - 1) * limit

	var comments []models.Comment
	var total int64
	s.db.Model(&models.Comment{}).Where("post_id = ?", postID).Count(&total)
	if err := s.db.Preload("Author").
		Where("post_id = ?", postID).
		Order("created_at ASC").
		Offset(offset).Limit(limit).
		Find(&comments).Error; err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch comments")
		return
	}

	result := make([]commentResponse, 0, len(comments))
	for _, cmt := range comments {
		var parentIDStr *string
		if cmt.ParentCommentID != nil {
			p := fmt.Sprintf("%d", *cmt.ParentCommentID)
			parentIDStr = &p
		}
		result = append(result, commentResponse{
			ID:              fmt.Sprintf("%d", cmt.ID),
			PostID:          toPostID(cmt.PostID),
			ParentCommentID: parentIDStr,
			AuthorID:        cmt.AuthorID,
			Author: userSummary{
				ID:        cmt.Author.ID,
				Username:  cmt.Author.Username,
				AvatarUrl: cmt.Author.ProfilePicture,
				Bio:       cmt.Author.Bio,
			},
			Content:   cmt.Content,
			CreatedAt: cmt.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, gin.H{keyData: result, keyPage: page, keyLimit: limit, keyTotal: total})
}

func (s *server) createComment(c *gin.Context) {
	identity, ok := shared.IdentityFromContext(c)
	if !ok {
		shared.AbortError(c, http.StatusUnauthorized, shared.ErrUnauthenticated, "authentication required")
		return
	}
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid post id")
		return
	}

	var req createCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid body")
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "content is required")
		return
	}

	comment := models.Comment{
		PostID:   postID,
		AuthorID: identity.UserID,
		Content:  req.Content,
	}
	if req.ParentCommentID != nil {
		parentID, parseErr := strconv.ParseInt(*req.ParentCommentID, 10, 64)
		if parseErr == nil {
			comment.ParentCommentID = &parentID
		}
	}
	if err := s.db.Create(&comment).Error; err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to create comment")
		return
	}
	s.db.Model(&models.Post{}).Where("id = ?", postID).UpdateColumn("comments_count", gorm.Expr("comments_count + 1"))

	var author models.User
	s.db.First(&author, "id = ?", identity.UserID)
	comment.Author = author

	var parentIDStr *string
	if comment.ParentCommentID != nil {
		str := fmt.Sprintf("%d", *comment.ParentCommentID)
		parentIDStr = &str
	}
	c.JSON(http.StatusCreated, commentResponse{
		ID:              fmt.Sprintf("%d", comment.ID),
		PostID:          toPostID(comment.PostID),
		ParentCommentID: parentIDStr,
		AuthorID:        comment.AuthorID,
		Author: userSummary{
			ID:        author.ID,
			Username:  author.Username,
			AvatarUrl: author.ProfilePicture,
			Bio:       author.Bio,
		},
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt.UTC().Format(time.RFC3339),
	})
}

func (s *server) searchPosts(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "q is required")
		return
	}
	page, limit := pageParams(c)
	offset := (page - 1) * limit
	caller, _ := shared.IdentityFromContext(c)

	pattern := "%" + q + "%"
	var posts []models.Post
	var total int64
	s.db.Model(&models.Post{}).
		Where("content ILIKE ? OR id IN (SELECT post_id FROM post_tags WHERE tag ILIKE ?)", pattern, pattern).
		Count(&total)
	if err := s.db.Preload("Author").
		Where("content ILIKE ? OR id IN (SELECT post_id FROM post_tags WHERE tag ILIKE ?)", pattern, pattern).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&posts).Error; err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to search posts")
		return
	}

	result := make([]postResponse, 0, len(posts))
	for _, p := range posts {
		result = append(result, buildPostResponse(s.db, p, caller.UserID))
	}
	c.JSON(http.StatusOK, gin.H{keyData: result, keyPage: page, keyLimit: limit, keyTotal: total})
}

func (s *server) getUserPosts(c *gin.Context) {
	targetUserID := strings.TrimSpace(c.Param("id"))
	page, limit := pageParams(c)
	offset := (page - 1) * limit
	caller, _ := shared.IdentityFromContext(c)

	var posts []models.Post
	var total int64
	s.db.Model(&models.Post{}).Where("author_id = ? AND parent_id IS NULL", targetUserID).Count(&total)
	if err := s.db.Preload("Author").
		Where("author_id = ? AND parent_id IS NULL", targetUserID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&posts).Error; err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch user posts")
		return
	}

	result := make([]postResponse, 0, len(posts))
	for _, p := range posts {
		result = append(result, buildPostResponse(s.db, p, caller.UserID))
	}
	c.JSON(http.StatusOK, gin.H{keyData: result, keyPage: page, keyLimit: limit, keyTotal: total})
}

// newRouter câble les routes sur le serveur. Extrait de main pour être testable
// (les tests d'intégration construisent le routeur sans démarrer le serveur).
func newRouter(srv *server) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.HandleMethodNotAllowed = true

	router.GET("/health", srv.health)
	router.POST("/internal/users", srv.internalUsers)
	router.GET("/feed", srv.getFeed)
	router.POST("/posts", srv.createPost)
	router.GET("/posts/search", srv.searchPosts)
	router.GET("/posts/:id", srv.getPost)
	router.DELETE("/posts/:id", srv.deletePost)
	router.POST("/posts/:id/like", srv.likePost)
	router.DELETE("/posts/:id/like", srv.unlikePost)
	router.GET("/posts/:id/comments", srv.getComments)
	router.POST("/posts/:id/comments", srv.createComment)
	router.GET("/users/:id/posts", srv.getUserPosts)
	router.NoMethod(func(c *gin.Context) {
		c.String(http.StatusMethodNotAllowed, "method not allowed")
	})

	return router
}
