package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type userService struct {
	db  *gorm.DB
	mc  *minio.Client
	cfg userConfig
}

// Erreurs métier internes, traduites en codes HTTP/erreur dans handler.go.
var (
	errValidation = errors.New("validation")
	errNotFound   = errors.New("not_found")
)

const roleUser = "user"

// Types de notifications émises par le user-service.
const (
	notifTypeFollow                = "follow"
	notifTypeFollowRequest         = "follow_request"
	notifTypeFollowRequestAccepted = "follow_request_accepted"
)

// followOutcome distingue, pour le handler, un abonnement direct (compte public)
// d'une simple demande en attente (compte privé), afin de répondre 204 vs 202.
type followOutcome int

const (
	outcomeFollowed  followOutcome = iota // abonnement effectif (créé ou déjà présent)
	outcomeRequested                      // demande d'abonnement en attente
)

// Noms des colonnes avatar/bannière, partagés entre les set*/clear*.
const (
	colAvatarURL       = "avatar_url"
	colAvatarObjectKey = "avatar_object_key"
	colBannerURL       = "banner_url"
	colBannerObjectKey = "banner_object_key"
)

func newUserService(db *gorm.DB, mc *minio.Client, cfg userConfig) *userService {
	return &userService{db: db, mc: mc, cfg: cfg}
}

// getProfile renvoie le profil cible, si l'appelant le suit, s'il a une demande
// en attente (compte privé), s'il a bloqué la cible, et si la cible l'a bloqué.
// gorm.ErrRecordNotFound remonte tel quel pour être traduit en 404 par le handler.
func (s *userService) getProfile(callerID, targetID string) (userModel, bool, bool, bool, bool, error) {
	var user userModel
	if err := s.db.First(&user, "id = ?", targetID).Error; err != nil {
		return userModel{}, false, false, false, false, err
	}

	if callerID == "" || callerID == targetID {
		return user, false, false, false, false, nil
	}

	var followCount, blockedByMeCount, hasBlockedMeCount int64
	if err := s.db.Model(&followModel{}).
		Where("follower_id = ? AND followee_id = ?", callerID, targetID).
		Count(&followCount).Error; err != nil {
		return userModel{}, false, false, false, false, err
	}
	if err := s.db.Model(&blockModel{}).
		Where("blocker_id = ? AND blocked_id = ?", callerID, targetID).
		Count(&blockedByMeCount).Error; err != nil {
		return userModel{}, false, false, false, false, err
	}
	if err := s.db.Model(&blockModel{}).
		Where("blocker_id = ? AND blocked_id = ?", targetID, callerID).
		Count(&hasBlockedMeCount).Error; err != nil {
		return userModel{}, false, false, false, false, err
	}

	followed := followCount > 0

	// Demande en attente pertinente uniquement sur un compte privé non encore suivi.
	requested := false
	if !followed && user.IsPrivate {
		var reqCount int64
		if err := s.db.Model(&followRequestModel{}).
			Where("requester_id = ? AND target_id = ?", callerID, targetID).
			Count(&reqCount).Error; err != nil {
			return userModel{}, false, false, false, false, err
		}
		requested = reqCount > 0
	}

	return user, followed, requested, blockedByMeCount > 0, hasBlockedMeCount > 0, nil
}

// canView indique si viewerID peut consulter les posts/re-posts de targetID, et
// renvoie aussi le flag isPrivate. Un compte public est toujours visible ; un
// compte privé ne l'est que pour lui-même ou ses abonnés. errNotFound si la cible
// n'existe pas. Utilisé par l'endpoint interne appelé par le post-service.
func (s *userService) canView(viewerID, targetID string) (isPrivate, visible bool, err error) {
	var target userModel
	if e := s.db.Select("id", "is_private").First(&target, "id = ?", targetID).Error; e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return false, false, errNotFound
		}
		return false, false, e
	}
	if !target.IsPrivate || viewerID == targetID {
		return target.IsPrivate, true, nil
	}
	if viewerID == "" {
		return true, false, nil
	}
	var count int64
	if e := s.db.Model(&followModel{}).
		Where("follower_id = ? AND followee_id = ?", viewerID, targetID).
		Count(&count).Error; e != nil {
		return true, false, e
	}
	return true, count > 0, nil
}

// hiddenAuthorIDs renvoie les IDs des comptes privés dont viewerID ne peut PAS voir
// les posts : tous les comptes privés sauf viewerID lui-même et ceux qu'il suit.
// Utilisé par le post-service pour exclure ces auteurs du feed global (GET /posts).
// Si viewerID est vide (appelant non authentifié), tous les comptes privés sont masqués.
// followingIDs renvoie les IDs des comptes que viewerID suit. Utilisé par le
// post-service pour appliquer le bonus « abonné » dans le feed Pour Toi.
func (s *userService) followingIDs(viewerID string) ([]string, error) {
	if viewerID == "" {
		return []string{}, nil
	}
	var ids []string
	if err := s.db.Model(&followModel{}).
		Where("follower_id = ?", viewerID).
		Pluck("followee_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// suggestedAuthorIDs interroge le post-service (endpoint interne) pour obtenir une
// liste d'auteurs classés par recouvrement de thèmes avec viewerID. Best-effort :
// erreur réseau / URL absente → renvoie nil (le handler bascule alors en repli).
func (s *userService) suggestedAuthorIDs(viewerID string, limit int) ([]string, error) {
	if s.cfg.PostURL == "" || viewerID == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// viewerID est un identifiant usr_<hex>, sûr en query string.
	url := fmt.Sprintf("%s/internal/posts/suggested-authors?viewer=%s&limit=%d", s.cfg.PostURL, viewerID, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build suggested-authors request: %w", err)
	}
	req.Header.Set("X-Internal-Key", s.cfg.InternalKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call suggested-authors: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("post-service suggested-authors: HTTP %d", resp.StatusCode)
	}
	var body struct {
		Authors []string `json:"authors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode suggested-authors: %w", err)
	}
	return body.Authors, nil
}

// rankSuggestions transforme une liste d'IDs candidats (theme-rankés) en profils
// à suggérer : exclut soi, les comptes déjà suivis et les blocages (deux sens),
// ne garde que les comptes actifs, puis complète avec des comptes récents si le
// signal thématique est insuffisant (nouvel arrivant, post-service injoignable).
func (s *userService) rankSuggestions(viewerID string, candidateIDs []string, want int) []userModel {
	exclude := map[string]bool{viewerID: true}
	if viewerID != "" {
		var following, blocked, blockedBy []string
		s.db.Model(&followModel{}).Where("follower_id = ?", viewerID).Pluck("followee_id", &following)
		s.db.Model(&blockModel{}).Where("blocker_id = ?", viewerID).Pluck("blocked_id", &blocked)
		s.db.Model(&blockModel{}).Where("blocked_id = ?", viewerID).Pluck("blocker_id", &blockedBy)
		for _, ids := range [][]string{following, blocked, blockedBy} {
			for _, id := range ids {
				exclude[id] = true
			}
		}
	}

	out := make([]userModel, 0, want)
	seen := map[string]bool{}

	// 1) Candidats theme-rankés, dans l'ordre fourni.
	for _, id := range candidateIDs {
		if len(out) >= want {
			break
		}
		if exclude[id] || seen[id] {
			continue
		}
		var u userModel
		if err := s.db.Where("id = ? AND status = ?", id, statusActive).First(&u).Error; err != nil {
			continue
		}
		seen[id] = true
		out = append(out, u)
	}

	// 2) Repli : complète avec des comptes récents actifs si pas assez de signal.
	if len(out) < want {
		var recent []userModel
		s.db.Where("status = ?", statusActive).Order("created_at DESC").Limit(want * 5).Find(&recent)
		for _, u := range recent {
			if len(out) >= want {
				break
			}
			if exclude[u.ID] || seen[u.ID] {
				continue
			}
			seen[u.ID] = true
			out = append(out, u)
		}
	}
	return out
}

func (s *userService) hiddenAuthorIDs(viewerID string) ([]string, error) {
	q := s.db.Model(&userModel{}).Where("is_private = ?", true)
	if viewerID != "" {
		q = q.Where("id <> ?", viewerID).
			Where("id NOT IN (?)", s.db.Model(&followModel{}).
				Select("followee_id").Where("follower_id = ?", viewerID))
	}
	var ids []string
	if err := q.Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// listBlocksPage est la requête paginée commune à listBlocked et listBlockedBy.
// joinCol est la colonne du JOIN qui référence users.id ; whereCol est la colonne
// filtrée par callerID.
func (s *userService) listBlocksPage(joinCol, whereCol, callerID string, page, limit int) ([]userModel, int64, error) {
	base := s.db.Model(&userModel{}).
		Joins("JOIN blocks ON blocks."+joinCol+" = users.id").
		Where("blocks."+whereCol+" = ?", callerID)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []userModel
	if err := base.Session(&gorm.Session{}).
		Order("blocks.created_at DESC").
		Offset((page - 1) * limit).Limit(limit).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// listBlockedBy renvoie les utilisateurs qui ont bloqué callerID.
func (s *userService) listBlockedBy(callerID string, page, limit int) ([]userModel, int64, error) {
	return s.listBlocksPage("blocker_id", "blocked_id", callerID, page, limit)
}

// blockUser crée la relation de blocage callerID → targetID. Idempotent.
func (s *userService) blockUser(callerID, targetID string) error {
	if callerID == targetID {
		return fmt.Errorf("%w: cannot block yourself", errValidation)
	}
	if err := s.ensureUserExists(targetID); err != nil {
		return err
	}
	return s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&blockModel{
		BlockerID: callerID,
		BlockedID: targetID,
	}).Error
}

// unblockUser supprime la relation de blocage. Idempotent.
func (s *userService) unblockUser(callerID, targetID string) error {
	if callerID == targetID {
		return fmt.Errorf("%w: cannot unblock yourself", errValidation)
	}
	return s.db.Where("blocker_id = ? AND blocked_id = ?", callerID, targetID).
		Delete(&blockModel{}).Error
}

// listBlocked renvoie les utilisateurs bloqués par callerID, du plus récent au plus ancien.
func (s *userService) listBlocked(callerID string, page, limit int) ([]userModel, int64, error) {
	return s.listBlocksPage("blocked_id", "blocker_id", callerID, page, limit)
}

// updateProfile applique un PATCH partiel (bio). Un champ fourni mais vide remet
// la colonne à NULL. Renvoie le profil rafraîchi. L'avatar n'est plus géré ici :
// il passe par uploadAvatarHandler (route PUT /users/me/avatar, upload vers MinIO).
func (s *userService) updateProfile(callerID string, req updateProfileRequest) (userModel, error) {
	updates := map[string]any{}
	// Pour chaque champ texte fourni : trim, et NULL si vide après trim.
	setText := func(col string, val *string) {
		if val == nil {
			return
		}
		if trimmed := strings.TrimSpace(*val); trimmed != "" {
			updates[col] = trimmed
		} else {
			updates[col] = nil
		}
	}
	setText("display_name", req.DisplayName)
	setText("pronouns", req.Pronouns)
	setText("bio", req.Bio)
	if req.IsPrivate != nil {
		updates["is_private"] = *req.IsPrivate
	}

	if len(updates) > 0 {
		res := s.db.Model(&userModel{}).Where("id = ?", callerID).Updates(updates)
		if res.Error != nil {
			return userModel{}, res.Error
		}
		if res.RowsAffected == 0 {
			return userModel{}, gorm.ErrRecordNotFound
		}
	}

	var user userModel
	if err := s.db.First(&user, "id = ?", callerID).Error; err != nil {
		return userModel{}, err
	}

	return user, nil
}

// setAvatar enregistre la nouvelle URL/clé d'avatar pour callerID et renvoie le
// profil rafraîchi ainsi que l'ancienne clé d'objet (nil si aucune), afin que
// l'appelant puisse supprimer l'ancien fichier dans MinIO. Le tout en transaction
// pour lire l'ancienne clé et écrire la nouvelle de façon cohérente.
func (s *userService) setAvatar(callerID, objectKey, url string) (userModel, *string, error) {
	var oldKey *string
	var user userModel

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&user, "id = ?", callerID).Error; err != nil {
			return err
		}
		oldKey = user.AvatarObjectKey

		return tx.Model(&userModel{}).Where("id = ?", callerID).
			Updates(map[string]any{colAvatarURL: url, colAvatarObjectKey: objectKey}).Error
	})
	if err != nil {
		return userModel{}, nil, fmt.Errorf("set avatar transaction: %w", err)
	}

	user.AvatarURL = &url
	user.AvatarObjectKey = &objectKey
	return user, oldKey, nil
}

// updateImageFields lit l'ancienne clé d'objet (via oldKeyOf) puis applique les
// mises à jour `updates` au profil de callerID, le tout en transaction pour lire
// et écrire de façon cohérente. Renvoie l'ancienne clé (nil si aucune) afin que
// l'appelant supprime l'ancien fichier dans MinIO ; `op` nomme l'opération dans
// le message d'erreur. Factorise clearAvatar, setBanner et clearBanner.
func (s *userService) updateImageFields(callerID, op string, oldKeyOf func(userModel) *string, updates map[string]any) (*string, error) {
	var oldKey *string

	err := s.db.Transaction(func(tx *gorm.DB) error {
		var user userModel
		if err := tx.First(&user, "id = ?", callerID).Error; err != nil {
			return err
		}
		oldKey = oldKeyOf(user)

		return tx.Model(&userModel{}).Where("id = ?", callerID).Updates(updates).Error
	})
	if err != nil {
		return nil, fmt.Errorf("%s transaction: %w", op, err)
	}
	return oldKey, nil
}

// clearAvatar remet l'avatar de callerID à NULL et renvoie l'ancienne clé d'objet
// (nil si aucune) pour que l'appelant supprime le fichier correspondant dans MinIO.
func (s *userService) clearAvatar(callerID string) (*string, error) {
	return s.updateImageFields(callerID, "clear avatar",
		func(u userModel) *string { return u.AvatarObjectKey },
		map[string]any{colAvatarURL: nil, colAvatarObjectKey: nil})
}

// setBanner enregistre la nouvelle URL/clé de bannière pour callerID et renvoie
// l'ancienne clé d'objet (nil si aucune), afin que l'appelant puisse supprimer
// l'ancien fichier dans MinIO. Pendant logique de setAvatar pour la bannière.
func (s *userService) setBanner(callerID, objectKey, url string) (*string, error) {
	return s.updateImageFields(callerID, "set banner",
		func(u userModel) *string { return u.BannerObjectKey },
		map[string]any{colBannerURL: url, colBannerObjectKey: objectKey})
}

// clearBanner remet la bannière de callerID à NULL et renvoie l'ancienne clé
// d'objet (nil si aucune) pour que l'appelant supprime le fichier dans MinIO.
func (s *userService) clearBanner(callerID string) (*string, error) {
	return s.updateImageFields(callerID, "clear banner",
		func(u userModel) *string { return u.BannerObjectKey },
		map[string]any{colBannerURL: nil, colBannerObjectKey: nil})
}

// createFollowTx insère la relation follower→followee et incrémente les compteurs
// dans la transaction tx. Renvoie created=false (sans erreur) si la relation
// existait déjà. Factorisé entre follow (compte public) et acceptFollowRequest.
func createFollowTx(tx *gorm.DB, followerID, followeeID string) (bool, error) {
	res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&followModel{
		FollowerID: followerID,
		FolloweeID: followeeID,
	})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		return false, nil // déjà abonné → idempotent, compteurs inchangés
	}
	if err := tx.Model(&userModel{}).Where("id = ?", followeeID).
		UpdateColumn("followers_count", gorm.Expr("followers_count + 1")).Error; err != nil {
		return false, err
	}
	if err := tx.Model(&userModel{}).Where("id = ?", followerID).
		UpdateColumn("following_count", gorm.Expr("following_count + 1")).Error; err != nil {
		return false, err
	}
	return true, nil
}

// follow gère l'abonnement de callerID à targetID. Sur un compte public, il crée
// directement la relation (outcomeFollowed). Sur un compte privé non encore suivi,
// il crée une demande en attente et notifie la cible (outcomeRequested). Idempotent
// dans les deux cas.
func (s *userService) follow(callerID, targetID string) (followOutcome, error) {
	if callerID == targetID {
		return outcomeFollowed, fmt.Errorf("%w: cannot follow yourself", errValidation)
	}

	var target userModel
	if err := s.db.Select("id", "is_private").First(&target, "id = ?", targetID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return outcomeFollowed, errNotFound
		}
		return outcomeFollowed, err
	}

	if target.IsPrivate {
		return s.requestFollow(callerID, targetID)
	}

	var created bool
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var e error
		created, e = createFollowTx(tx, callerID, targetID)
		return e
	})
	if err != nil {
		return outcomeFollowed, fmt.Errorf("follow transaction: %w", err)
	}
	if created {
		s.sendNotifAsync(targetID, callerID, notifTypeFollow)
		s.syncSubscriptionAsync(callerID, targetID, true)
	}
	return outcomeFollowed, nil
}

// requestFollow gère l'abonnement à un compte privé. Si callerID est déjà abonné
// (ex. abonnement antérieur au passage en privé), c'est un no-op « abonné ». Sinon
// il crée une demande en attente (idempotente) et notifie la cible.
func (s *userService) requestFollow(callerID, targetID string) (followOutcome, error) {
	var followCount int64
	if err := s.db.Model(&followModel{}).
		Where("follower_id = ? AND followee_id = ?", callerID, targetID).
		Count(&followCount).Error; err != nil {
		return outcomeRequested, err
	}
	if followCount > 0 {
		return outcomeFollowed, nil
	}

	res := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&followRequestModel{
		RequesterID: callerID,
		TargetID:    targetID,
	})
	if res.Error != nil {
		return outcomeRequested, fmt.Errorf("follow request: %w", res.Error)
	}
	if res.RowsAffected > 0 {
		s.sendNotifAsync(targetID, callerID, notifTypeFollowRequest)
	}
	return outcomeRequested, nil
}

// acceptFollowRequest valide la demande requesterID → targetID : crée l'abonnement
// et supprime la demande dans une même transaction, puis notifie le demandeur
// (follow_request_accepted) et met à jour l'abonnement notif. errNotFound si
// aucune demande n'est en attente.
func (s *userService) acceptFollowRequest(targetID, requesterID string) error {
	var created bool
	err := s.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("requester_id = ? AND target_id = ?", requesterID, targetID).
			Delete(&followRequestModel{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errNotFound
		}
		var e error
		created, e = createFollowTx(tx, requesterID, targetID)
		return e
	})
	if err != nil {
		return fmt.Errorf("accept follow request: %w", err)
	}
	s.sendNotifAsync(requesterID, targetID, notifTypeFollowRequestAccepted)
	if created {
		s.syncSubscriptionAsync(requesterID, targetID, true)
	}
	return nil
}

// rejectFollowRequest supprime la demande requesterID → targetID. Idempotent (pas
// d'erreur si elle n'existe pas) et, volontairement, n'envoie aucune notification.
func (s *userService) rejectFollowRequest(targetID, requesterID string) error {
	return s.db.Where("requester_id = ? AND target_id = ?", requesterID, targetID).
		Delete(&followRequestModel{}).Error
}

// listFollowRequests renvoie (une page des) profils des demandeurs en attente vers
// targetID, du plus récent au plus ancien, ainsi que le total.
func (s *userService) listFollowRequests(targetID string, page, limit int) ([]userModel, int64, error) {
	base := s.db.Model(&userModel{}).
		Joins("JOIN follow_requests ON follow_requests.requester_id = users.id").
		Where("follow_requests.target_id = ?", targetID)

	return paginateUsersOrdered(base, page, limit, "follow_requests.created_at DESC")
}

func (s *userService) syncSubscriptionAsync(followerID, followeeID string, subscribe bool) {
	if s.cfg.NotifURL == "" {
		return
	}
	go func() {
		method := http.MethodPost
		if !subscribe {
			method = http.MethodDelete
		}
		data, _ := json.Marshal(map[string]string{
			"subscriber_id":  followerID,
			"target_user_id": followeeID,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, method, s.cfg.NotifURL+"/internal/subscriptions", bytes.NewReader(data))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Key", s.cfg.InternalKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("syncSubscriptionAsync: %v", err)
			return
		}
		if resp.StatusCode >= 300 {
			log.Printf("syncSubscriptionAsync: unexpected status %s", resp.Status)
		}
		_ = resp.Body.Close()
	}()
}

// sendNotifAsync envoie (best-effort) une notification à recipientID, déclenchée
// par actorID, du type donné (follow, follow_request, follow_request_accepted).
// L'entité pointée est l'acteur lui-même (un utilisateur), de sorte qu'un clic mène
// vers son profil.
func (s *userService) sendNotifAsync(recipientID, actorID, notifType string) {
	if s.cfg.NotifURL == "" {
		return
	}
	go func() {
		var actor userModel
		if err := s.db.Select("username").Where("id = ?", actorID).First(&actor).Error; err != nil {
			log.Printf("sendNotifAsync: fetch actor: %v", err)
		}
		data, _ := json.Marshal(map[string]string{
			"user_id":        recipientID,
			"type":           notifType,
			"actor_id":       actorID,
			"actor_username": actor.Username,
			"entity_id":      actorID,
			"entity_type":    roleUser,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.NotifURL+"/internal/notifications", bytes.NewReader(data))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Key", s.cfg.InternalKey)
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

// unfollow supprime la relation callerID → targetID. Idempotent : si elle n'existe
// pas, renvoie nil. Les compteurs ne sont décrémentés que si une ligne a été
// supprimée, et jamais en dessous de 0.
func (s *userService) unfollow(callerID, targetID string) error {
	if callerID == targetID {
		return fmt.Errorf("%w: cannot unfollow yourself", errValidation)
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Annule aussi une éventuelle demande en attente (compte privé) : le bouton
		// « Demandé » repasse alors à « Suivre ».
		if err := tx.Where("requester_id = ? AND target_id = ?", callerID, targetID).
			Delete(&followRequestModel{}).Error; err != nil {
			return err
		}
		res := tx.Where("follower_id = ? AND followee_id = ?", callerID, targetID).
			Delete(&followModel{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // pas abonné → idempotent
		}

		if err := tx.Model(&userModel{}).Where("id = ? AND followers_count > 0", targetID).
			UpdateColumn("followers_count", gorm.Expr("followers_count - 1")).Error; err != nil {
			return err
		}
		return tx.Model(&userModel{}).Where("id = ? AND following_count > 0", callerID).
			UpdateColumn("following_count", gorm.Expr("following_count - 1")).Error
	})
	if err != nil {
		return fmt.Errorf("unfollow transaction: %w", err)
	}
	s.syncSubscriptionAsync(callerID, targetID, false)
	return nil
}

// createProfile crée le profil détenu par le user-service, avec l'ID fourni par
// l'auth-service. Idempotent sur l'ID (ON CONFLICT DO NOTHING) pour que les
// retries de l'auth soient sans danger.
func (s *userService) createProfile(id, username, email, role string) error {
	id = strings.TrimSpace(id)
	username = strings.ToLower(strings.TrimSpace(username))
	email = strings.ToLower(strings.TrimSpace(email))
	role = strings.TrimSpace(role)

	if id == "" || username == "" || email == "" {
		return fmt.Errorf("%w: id, username and email are required", errValidation)
	}
	if role == "" {
		role = roleUser
	}

	// displayName par défaut = identifiant (le username, déjà sans "@") ; l'utilisateur
	// pourra le personnaliser ensuite via updateProfile.
	defaultDisplayName := username

	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&userModel{
		ID:          id,
		Username:    username,
		Email:       email,
		Role:        role,
		DisplayName: &defaultDisplayName,
	}).Error
}

func (s *userService) ensureUserExists(id string) error {
	var count int64
	if err := s.db.Model(&userModel{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errNotFound
	}
	return nil
}

// getUserByUsername renvoie le profil dont le username correspond (insensible à la casse).
func (s *userService) getUserByUsername(username string) (userModel, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	var user userModel
	if err := s.db.First(&user, "username = ?", username).Error; err != nil {
		return userModel{}, err
	}
	return user, nil
}

// listFollowing renvoie (page de) utilisateurs que targetID suit, du plus récent
// au plus ancien, ainsi que le total. errNotFound si targetID n'existe pas.
func (s *userService) listFollowing(targetID string, page, limit int) ([]userModel, int64, error) {
	if err := s.ensureUserExists(targetID); err != nil {
		return nil, 0, err
	}

	base := s.db.Model(&userModel{}).
		Joins("JOIN follows ON follows.followee_id = users.id").
		Where("follows.follower_id = ?", targetID)

	return paginateUsers(base, page, limit)
}

// listFollowers renvoie (page de) abonnés de targetID.
func (s *userService) listFollowers(targetID string, page, limit int) ([]userModel, int64, error) {
	if err := s.ensureUserExists(targetID); err != nil {
		return nil, 0, err
	}

	base := s.db.Model(&userModel{}).
		Joins("JOIN follows ON follows.follower_id = users.id").
		Where("follows.followee_id = ?", targetID)

	return paginateUsers(base, page, limit)
}

// listUsers renvoie tous les utilisateurs paginés, du plus récent au plus ancien.
func (s *userService) listUsers(page, limit int, q string) ([]userModel, int64, error) {
	base := s.db.Model(&userModel{})
	// On accepte un '@' initial (réflexe courant) et on cherche aussi bien sur le
	// username que sur le displayName, qui est le nom affiché un peu partout dans l'UI.
	if q = strings.TrimPrefix(strings.TrimSpace(q), "@"); q != "" {
		pattern := "%" + q + "%"
		base = base.Where("username ILIKE ? OR display_name ILIKE ?", pattern, pattern)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []userModel
	if err := base.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// batchFollowedByMe renvoie l'ensemble des IDs que callerID suit parmi la liste fournie.
func (s *userService) batchFollowedByMe(callerID string, users []userModel) map[string]bool {
	result := map[string]bool{}
	if callerID == "" || len(users) == 0 {
		return result
	}
	ids := make([]string, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	var follows []followModel
	s.db.Where("follower_id = ? AND followee_id IN ?", callerID, ids).Find(&follows)
	for _, f := range follows {
		result[f.FolloweeID] = true
	}
	return result
}

// ── Sanctions (modération) ────────────────────────────────────────────────────

const (
	statusActive    = "active"
	statusSuspended = "suspended"
	statusBanned    = "banned"

	sanctionTypeBan        = "ban"
	sanctionTypeSuspension = "suspension"
)

// sanctionUser applique une sanction à targetID, met le compte dans l'état
// correspondant (banned/suspended) et clôt toute sanction active précédente,
// le tout en transaction. Propage ensuite l'état à l'auth-service (best-effort).
func (s *userService) sanctionUser(callerID, targetID string, req sanctionRequest) (sanctionModel, error) {
	if callerID == targetID {
		return sanctionModel{}, fmt.Errorf("%w: cannot sanction yourself", errValidation)
	}

	var status string
	switch req.Type {
	case sanctionTypeBan:
		status = statusBanned
	case sanctionTypeSuspension:
		status = statusSuspended
	default:
		return sanctionModel{}, fmt.Errorf("%w: type must be ban or suspension", errValidation)
	}

	var reason *string
	if req.Reason != nil {
		if r := strings.TrimSpace(*req.Reason); r != "" {
			if len(r) > 255 {
				r = r[:255]
			}
			reason = &r
		}
	}

	now := time.Now().UTC()
	sanction := sanctionModel{UserID: targetID, ModeratorID: &callerID, Type: req.Type, Reason: reason, StartedAt: now}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&sanctionModel{}).Where("user_id = ? AND ended_at IS NULL", targetID).
			Update("ended_at", now).Error; err != nil {
			return err
		}
		if err := tx.Create(&sanction).Error; err != nil {
			return err
		}
		res := tx.Model(&userModel{}).Where("id = ?", targetID).Update(keyStatus, status)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errNotFound
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errNotFound) {
			return sanctionModel{}, errNotFound
		}
		return sanctionModel{}, fmt.Errorf("sanction transaction: %w", err)
	}
	s.syncAuthStatusAsync(targetID, status)
	return sanction, nil
}

// liftSanction lève la sanction active de targetID et réactive le compte.
func (s *userService) liftSanction(targetID string) error {
	if err := s.ensureUserExists(targetID); err != nil {
		return err
	}
	now := time.Now().UTC()
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&sanctionModel{}).Where("user_id = ? AND ended_at IS NULL", targetID).
			Update("ended_at", now).Error; err != nil {
			return err
		}
		return tx.Model(&userModel{}).Where("id = ?", targetID).Update(keyStatus, statusActive).Error
	})
	if err != nil {
		return fmt.Errorf("lift sanction transaction: %w", err)
	}
	s.syncAuthStatusAsync(targetID, statusActive)
	return nil
}

// listSanctions renvoie l'historique des sanctions de targetID (plus récent d'abord).
func (s *userService) listSanctions(targetID string) ([]sanctionModel, error) {
	if err := s.ensureUserExists(targetID); err != nil {
		return nil, err
	}
	var sanctions []sanctionModel
	if err := s.db.Where("user_id = ?", targetID).Order("created_at DESC").Find(&sanctions).Error; err != nil {
		return nil, err
	}
	return sanctions, nil
}

// syncAuthStatusAsync propage l'état de modération à l'auth-service afin de
// bloquer (ou rouvrir) l'accès au compte. Best-effort : une erreur est
// tracée mais ne fait pas échouer la sanction déjà persistée.
func (s *userService) syncAuthStatusAsync(userID, status string) {
	if s.cfg.AuthURL == "" {
		return
	}
	go func() {
		data, _ := json.Marshal(map[string]string{keyStatus: status})
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, s.cfg.AuthURL+"/internal/users/"+userID+"/status", bytes.NewReader(data))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Internal-Key", s.cfg.InternalKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("syncAuthStatusAsync: %v", err)
			return
		}
		if resp.StatusCode >= 300 {
			log.Printf("syncAuthStatusAsync: unexpected status %s", resp.Status)
		}
		_ = resp.Body.Close()
	}()
}

// paginateUsers compte le total puis récupère la page demandée, ordonnée par date
// d'abonnement (follows.created_at).
func paginateUsers(base *gorm.DB, page, limit int) ([]userModel, int64, error) {
	return paginateUsersOrdered(base, page, limit, "follows.created_at DESC")
}

// paginateUsersOrdered factorise count + page avec une clause ORDER explicite.
// Session() isole chaque requête pour que le Count (sans ORDER) ne pollue pas le
// Find et inversement.
func paginateUsersOrdered(base *gorm.DB, page, limit int, order string) ([]userModel, int64, error) {
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var users []userModel
	if err := base.Session(&gorm.Session{}).
		Order(order).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}
