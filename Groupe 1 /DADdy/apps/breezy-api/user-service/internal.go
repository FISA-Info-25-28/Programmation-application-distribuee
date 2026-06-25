package main

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"daddy/apps/breezy-api/internal/shared"
)

// requireInternalKey protège les routes internes service-à-service. La clé est
// comparée en temps constant. Si aucune clé n'est configurée, tout est refusé
// (on n'ouvre jamais l'endpoint sans authentification).
func requireInternalKey(expected string) gin.HandlerFunc {
	expectedBytes := []byte(expected)
	return func(c *gin.Context) {
		provided := []byte(c.GetHeader("X-Internal-Key"))
		if len(expectedBytes) == 0 || subtle.ConstantTimeCompare(provided, expectedBytes) != 1 {
			shared.AbortError(c, http.StatusForbidden, shared.ErrForbidden, "forbidden")
			return
		}
		c.Next()
	}
}

// resolveUsernamesHandler résout une liste de usernames → [{id, username}].
// Utilisé par le post-service pour identifier les utilisateurs mentionnés via @username
// qui n'ont pas été sélectionnés depuis le dropdown d'autocomplétion.
func (s *userService) resolveUsernamesHandler(c *gin.Context) {
	var body struct {
		Usernames []string `json:"usernames" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "usernames required")
		return
	}
	if len(body.Usernames) == 0 {
		c.JSON(http.StatusOK, gin.H{keyData: []struct{}{}})
		return
	}
	lower := make([]string, len(body.Usernames))
	for i, u := range body.Usernames {
		lower[i] = strings.ToLower(strings.TrimSpace(u))
	}
	var users []userModel
	s.db.Select("id, username").Where("username IN ?", lower).Find(&users)
	type entry struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	result := make([]entry, len(users))
	for i, u := range users {
		result[i] = entry{ID: u.ID, Username: u.Username}
	}
	c.JSON(http.StatusOK, gin.H{keyData: result})
}

func (s *userService) createInternalUserHandler(c *gin.Context) {
	var req internalCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	if err := s.createProfile(req.ID, req.Username, req.Email, req.Role); err != nil {
		if errors.Is(err, errValidation) {
			shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, err.Error())
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to create user profile")
		return
	}

	c.Status(http.StatusCreated)
}

// batchAvatarsHandler répond à POST /internal/users/batch-avatars, appelé par le
// post-service pour récupérer les URLs d'avatar des auteurs d'un feed en une seule requête.
func (s *userService) batchAvatarsHandler(c *gin.Context) {
	var body struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "ids required")
		return
	}
	if len(body.IDs) == 0 {
		c.JSON(http.StatusOK, gin.H{keyData: []struct{}{}})
		return
	}
	var users []userModel
	s.db.Select("id, username, avatar_url").Where("id IN ?", body.IDs).Find(&users)
	type entry struct {
		ID        string  `json:"id"`
		Username  string  `json:"username"`
		AvatarURL *string `json:"avatarUrl"`
	}
	result := make([]entry, len(users))
	for i, u := range users {
		result[i] = entry{ID: u.ID, Username: u.Username, AvatarURL: u.AvatarURL}
	}
	c.JSON(http.StatusOK, gin.H{keyData: result})
}

// canViewHandler répond à GET /internal/users/:id/can-view?viewer=<id>, appelé par
// le post-service pour décider s'il peut servir les posts/re-posts d'un compte. Le
// viewer est passé en query (et non via X-User-Id) car cet appel est service-à-service.
func (s *userService) canViewHandler(c *gin.Context) {
	targetID := strings.TrimSpace(c.Param("id"))
	viewerID := strings.TrimSpace(c.Query("viewer"))
	if targetID == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "user id is required")
		return
	}

	isPrivate, visible, err := s.canView(viewerID, targetID)
	if err != nil {
		if errors.Is(err, errNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to check visibility")
		return
	}

	c.JSON(http.StatusOK, gin.H{"isPrivate": isPrivate, "canView": visible})
}

// followingIDsHandler répond à GET /internal/users/:id/following-ids, appelé par
// le post-service pour appliquer le bonus « abonné » dans le feed Pour Toi.
// Renvoie la liste des IDs que l'utilisateur :id suit.
func (s *userService) followingIDsHandler(c *gin.Context) {
	viewerID := strings.TrimSpace(c.Param("id"))
	ids, err := s.followingIDs(viewerID)
	if err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch following")
		return
	}
	if ids == nil {
		ids = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"following": ids})
}

// hiddenAuthorsHandler répond à GET /internal/users/hidden-authors?viewer=<id>,
// appelé par le post-service pour filtrer le feed global. Renvoie la liste des IDs
// de comptes privés que le viewer ne peut pas voir (non abonné), à exclure des posts.
func (s *userService) hiddenAuthorsHandler(c *gin.Context) {
	viewerID := strings.TrimSpace(c.Query("viewer"))

	ids, err := s.hiddenAuthorIDs(viewerID)
	if err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to compute hidden authors")
		return
	}
	if ids == nil {
		ids = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"hidden": ids})
}
