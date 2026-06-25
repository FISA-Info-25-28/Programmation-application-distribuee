package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"daddy/apps/breezy-api/internal/shared"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
	keyData      = "data"
	keyPage      = "page"
	keyLimit     = "limit"
	keyTotal     = "total"
	keyStatus    = "status"
)

// parsePagination lit page/limit depuis la query string, avec valeurs par défaut
// et plafond sur limit pour éviter les requêtes abusives.
func parsePagination(c *gin.Context) (page, limit int) {
	page = defaultPage
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = v
	}
	limit = defaultLimit
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
		limit = v
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

// toUserResponse mappe le modèle vers le schéma OpenAPI.
//   - includeEmail : true uniquement quand l'appelant consulte son propre profil.
//   - followedByMe / blockedByMe / hasBlockedMe : nil → champ omis (ex: PATCH /users/me).
func toUserResponse(u userModel, includeEmail bool, followedByMe *bool, blockedByMe *bool, hasBlockedMe *bool) userResponse {
	resp := userResponse{
		ID:             u.ID,
		Username:       u.Username,
		DisplayName:    u.DisplayName,
		Pronouns:       u.Pronouns,
		Bio:            u.Bio,
		AvatarURL:      u.AvatarURL,
		BannerURL:      u.BannerURL,
		Role:           u.Role,
		IsPrivate:      u.IsPrivate,
		Status:         u.Status,
		FollowersCount: u.FollowersCount,
		FollowingCount: u.FollowingCount,
		IsFollowedByMe: followedByMe,
		IsBlockedByMe:  blockedByMe,
		HasBlockedMe:   hasBlockedMe,
		CreatedAt:      u.CreatedAt.UTC().Format(time.RFC3339),
	}
	if includeEmail {
		email := u.Email
		resp.Email = &email
	}
	return resp
}

func (s *userService) getUserByUsernameHandler(c *gin.Context) {
	username := strings.TrimSpace(c.Query("username"))
	if username == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "username query param is required")
		return
	}

	identity, _ := shared.IdentityFromContext(c)
	user, err := s.getUserByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch user")
		return
	}

	isSelf := user.ID == identity.UserID
	c.JSON(http.StatusOK, toUserResponse(user, isSelf, nil, nil, nil))
}

func (s *userService) getProfileHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)

	targetID := strings.TrimSpace(c.Param("id"))
	if targetID == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "user id is required")
		return
	}

	user, followed, requested, blocked, hasBlockedMe, err := s.getProfile(identity.UserID, targetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch user")
		return
	}

	isSelf := user.ID == identity.UserID
	resp := toUserResponse(user, isSelf, &followed, &blocked, &hasBlockedMe)
	canView := isSelf || !user.IsPrivate || followed
	resp.CanViewPosts = &canView
	resp.FollowRequested = &requested
	c.JSON(http.StatusOK, resp)
}

func (s *userService) updateProfileHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}
	if err := validateProfileUpdate(req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, err.Error())
		return
	}

	user, err := s.updateProfile(identity.UserID, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to update profile")
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user, true, nil, nil, nil))
}

func (s *userService) followHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)

	targetID := strings.TrimSpace(c.Param("id"))
	if targetID == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "user id is required")
		return
	}

	outcome, err := s.follow(identity.UserID, targetID)
	switch {
	case errors.Is(err, errValidation):
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "cannot follow yourself")
	case errors.Is(err, errNotFound):
		shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
	case err != nil:
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to follow user")
	case outcome == outcomeRequested:
		// Compte privé : demande créée, en attente d'acceptation.
		c.JSON(http.StatusAccepted, gin.H{keyStatus: "requested"})
	default:
		c.Status(http.StatusNoContent)
	}
}

// listFollowRequestsHandler renvoie les requêtes d'abonnement en attente vers
// l'appelant (profils des demandeurs).
func (s *userService) listFollowRequestsHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)
	page, limit := parsePagination(c)

	users, total, err := s.listFollowRequests(identity.UserID, page, limit)
	if err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch follow requests")
		return
	}

	responses := make([]userResponse, 0, len(users))
	for _, u := range users {
		responses = append(responses, toUserResponse(u, false, nil, nil, nil))
	}
	c.JSON(http.StatusOK, gin.H{keyData: responses, keyPage: page, keyLimit: limit, keyTotal: total})
}

// requireIDParam extrait le param :id, le valide non vide et écrit une 400 sinon.
// Renvoie (id, true) si présent, ("", false) si la réponse d'erreur a déjà été émise.
func requireIDParam(c *gin.Context) (string, bool) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "id is required")
		return "", false
	}
	return id, true
}

// acceptFollowRequestHandler accepte la demande dont le demandeur est :id.
func (s *userService) acceptFollowRequestHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)
	requesterID, ok := requireIDParam(c)
	if !ok {
		return
	}

	err := s.acceptFollowRequest(identity.UserID, requesterID)
	switch {
	case errors.Is(err, errNotFound):
		shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "follow request not found")
	case err != nil:
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to accept follow request")
	default:
		c.Status(http.StatusNoContent)
	}
}

// rejectFollowRequestHandler refuse (supprime) la demande dont le demandeur est :id.
// Idempotent et sans notification.
func (s *userService) rejectFollowRequestHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)
	requesterID, ok := requireIDParam(c)
	if !ok {
		return
	}

	if err := s.rejectFollowRequest(identity.UserID, requesterID); err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to reject follow request")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *userService) unfollowHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)

	targetID, ok := requireIDParam(c)
	if !ok {
		return
	}

	err := s.unfollow(identity.UserID, targetID)
	switch {
	case errors.Is(err, errValidation):
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "cannot unfollow yourself")
	case err != nil:
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to unfollow user")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (s *userService) listFollowingHandler(c *gin.Context) {
	s.respondUserList(c, s.listFollowing)
}

func (s *userService) listFollowersHandler(c *gin.Context) {
	s.respondUserList(c, s.listFollowers)
}

// ── Sanctions (modération, réservé au staff) ──────────────────────────────────

func toSanctionResponse(m sanctionModel) sanctionResponse {
	var ended *string
	if m.EndedAt != nil {
		e := m.EndedAt.UTC().Format(time.RFC3339)
		ended = &e
	}
	return sanctionResponse{
		ID:          m.ID,
		UserID:      m.UserID,
		ModeratorID: m.ModeratorID,
		Type:        m.Type,
		Reason:      m.Reason,
		StartedAt:   m.StartedAt.UTC().Format(time.RFC3339),
		EndedAt:     ended,
		CreatedAt:   m.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *userService) sanctionUserHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)
	targetID := strings.TrimSpace(c.Param("id"))
	if targetID == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "user id is required")
		return
	}

	var req sanctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "invalid request body")
		return
	}

	sanction, err := s.sanctionUser(identity.UserID, targetID, req)
	switch {
	case errors.Is(err, errValidation):
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, err.Error())
	case errors.Is(err, errNotFound):
		shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
	case err != nil:
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to sanction user")
	default:
		c.JSON(http.StatusCreated, toSanctionResponse(sanction))
	}
}

func (s *userService) liftSanctionHandler(c *gin.Context) {
	targetID := strings.TrimSpace(c.Param("id"))
	if targetID == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "user id is required")
		return
	}

	err := s.liftSanction(targetID)
	switch {
	case errors.Is(err, errNotFound):
		shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
	case err != nil:
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to lift sanction")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (s *userService) listSanctionsHandler(c *gin.Context) {
	targetID := strings.TrimSpace(c.Param("id"))
	if targetID == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "user id is required")
		return
	}

	sanctions, err := s.listSanctions(targetID)
	if err != nil {
		if errors.Is(err, errNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch sanctions")
		return
	}

	out := make([]sanctionResponse, 0, len(sanctions))
	for _, m := range sanctions {
		out = append(out, toSanctionResponse(m))
	}
	c.JSON(http.StatusOK, gin.H{keyData: out})
}

func (s *userService) listUsersHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)
	page, limit := parsePagination(c)
	q := strings.TrimSpace(c.Query("q"))

	users, total, err := s.listUsers(page, limit, q)
	if err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch users")
		return
	}

	followedIDs := s.batchFollowedByMe(identity.UserID, users)
	responses := make([]userResponse, 0, len(users))
	for _, u := range users {
		f := followedIDs[u.ID]
		isSelf := u.ID == identity.UserID
		responses = append(responses, toUserResponse(u, isSelf, &f, nil, nil))
	}

	c.JSON(http.StatusOK, gin.H{keyData: responses, keyPage: page, keyLimit: limit, keyTotal: total})
}

// suggestionsHandler répond à GET /users/suggestions : comptes à suivre, classés
// par recouvrement de thèmes avec le lecteur (signal interne, jamais exposé),
// hors soi / déjà-suivis / blocages. Repli sur des comptes récents si le signal
// thématique est insuffisant.
func (s *userService) suggestionsHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)
	const want = 5
	candidates, _ := s.suggestedAuthorIDs(identity.UserID, 30) // erreur → nil → repli
	users := s.rankSuggestions(identity.UserID, candidates, want)

	notFollowed := false
	responses := make([]userResponse, 0, len(users))
	for _, u := range users {
		responses = append(responses, toUserResponse(u, false, &notFollowed, nil, nil))
	}
	c.JSON(http.StatusOK, gin.H{keyData: responses, keyPage: 1, keyLimit: want, keyTotal: len(responses)})
}

// respondUserList factorise le squelette commun à /following et /followers.
func (s *userService) respondUserList(c *gin.Context, fetch func(string, int, int) ([]userModel, int64, error)) {
	identity, _ := shared.IdentityFromContext(c)
	targetID := strings.TrimSpace(c.Param("id"))
	if targetID == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "user id is required")
		return
	}

	page, limit := parsePagination(c)

	users, total, err := fetch(targetID, page, limit)
	if err != nil {
		if errors.Is(err, errNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
			shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
			return
		}
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch users")
		return
	}

	followedIDs := s.batchFollowedByMe(identity.UserID, users)
	responses := make([]userResponse, 0, len(users))
	for _, u := range users {
		f := followedIDs[u.ID]
		responses = append(responses, toUserResponse(u, false, &f, nil, nil))
	}

	c.JSON(http.StatusOK, gin.H{keyData: responses, keyPage: page, keyLimit: limit, keyTotal: total})
}

func (s *userService) blockHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)
	targetID := strings.TrimSpace(c.Param("id"))
	if targetID == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "user id is required")
		return
	}
	err := s.blockUser(identity.UserID, targetID)
	switch {
	case errors.Is(err, errValidation):
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "cannot block yourself")
	case errors.Is(err, errNotFound):
		shared.AbortError(c, http.StatusNotFound, shared.ErrNotFound, "user not found")
	case err != nil:
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to block user")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (s *userService) unblockHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)
	targetID := strings.TrimSpace(c.Param("id"))
	if targetID == "" {
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "user id is required")
		return
	}
	err := s.unblockUser(identity.UserID, targetID)
	switch {
	case errors.Is(err, errValidation):
		shared.AbortError(c, http.StatusBadRequest, shared.ErrValidation, "cannot unblock yourself")
	case err != nil:
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to unblock user")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (s *userService) listBlockedHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)
	page, limit := parsePagination(c)
	users, total, err := s.listBlocked(identity.UserID, page, limit)
	if err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch blocked users")
		return
	}
	t := true
	responses := make([]userResponse, 0, len(users))
	for _, u := range users {
		responses = append(responses, toUserResponse(u, false, nil, &t, nil))
	}
	c.JSON(http.StatusOK, gin.H{keyData: responses, keyPage: page, keyLimit: limit, keyTotal: total})
}

func (s *userService) listBlockedByHandler(c *gin.Context) {
	identity, _ := shared.IdentityFromContext(c)
	page, limit := parsePagination(c)
	users, total, err := s.listBlockedBy(identity.UserID, page, limit)
	if err != nil {
		shared.AbortError(c, http.StatusInternalServerError, shared.ErrInternal, "failed to fetch blockers")
		return
	}
	t := true
	responses := make([]userResponse, 0, len(users))
	for _, u := range users {
		responses = append(responses, toUserResponse(u, false, nil, nil, &t))
	}
	c.JSON(http.StatusOK, gin.H{keyData: responses, keyPage: page, keyLimit: limit, keyTotal: total})
}
