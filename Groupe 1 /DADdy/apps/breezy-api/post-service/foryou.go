package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	neturl "net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Poids des composantes du score « Pour Toi ». Ajustables ici sans toucher à la
// requête SQL. Les trois premiers pondèrent les préférences organiques
// (user_affinity), les suivants assurent fraîcheur et découverte.
const (
	wHash           = 3.0
	wTheme          = 3.0
	wAuthor         = 2.0
	wFollow         = 2.0
	wPop            = 1.0
	wRecency        = 2.0
	recencyTauHours = 48.0
)

// forYouFeed sert le feed personnalisé. Le score d'un post combine l'affinité du
// lecteur pour ses hashtags / son thème / son auteur (préférences organiques
// matérialisées dans user_affinity), un bonus si l'auteur est suivi, la
// popularité globale et une décroissance temporelle. Pour un visiteur anonyme ou
// un nouvel arrivant sans historique, on retombe sur le feed global (récence)
// afin de ne jamais servir un feed vide.
func (a *app) forYouFeed(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	callerID := a.callerID(c)
	if callerID == "" {
		a.globalFeed(c)
		return
	}

	var affCount int64
	a.db.Model(&userAffinityModel{}).Where("user_id = ?", callerID).Count(&affCount)
	if affCount == 0 {
		a.globalFeed(c)
		return
	}

	hidden := a.hiddenAuthors(callerID)
	followed := a.followingIDs(callerID)

	// Total = ensemble des candidats (posts racine visibles), indépendant du
	// scoring : sert juste à la pagination côté client.
	countQ := a.db.Model(&postModel{}).Where("parent_id IS NULL")
	if len(hidden) > 0 {
		countQ = countQ.Where("author_id NOT IN ?", hidden)
	}
	var total int64
	countQ.Count(&total)

	// L'affinité hashtag d'un post = somme des poids du lecteur pour chacun de ses
	// hashtags ; thème et auteur sont des jointures directes sur user_affinity.
	sql := `SELECT posts.* FROM posts
LEFT JOIN (
    SELECT ph.post_id, SUM(ua.weight) AS s
    FROM post_hashtags ph
    JOIN hashtags h ON h.id = ph.hashtag_id
    JOIN user_affinity ua ON ua.kind = 'hashtag' AND ua.ref = h.name AND ua.user_id = ?
    GROUP BY ph.post_id
) ha ON ha.post_id = posts.id
LEFT JOIN user_affinity ta ON ta.user_id = ? AND ta.kind = 'theme' AND ta.ref = posts.theme
LEFT JOIN user_affinity aa ON aa.user_id = ? AND aa.kind = 'author' AND aa.ref = posts.author_id
WHERE posts.parent_id IS NULL`
	args := []any{callerID, callerID, callerID}
	if len(hidden) > 0 {
		sql += " AND posts.author_id NOT IN ?"
		args = append(args, hidden)
	}

	followExpr := "0"
	if len(followed) > 0 {
		followExpr = "CASE WHEN posts.author_id IN ? THEN 1 ELSE 0 END"
	}
	orderExpr := fmt.Sprintf(
		"%g*COALESCE(ha.s,0) + %g*COALESCE(ta.weight,0) + %g*COALESCE(aa.weight,0)"+
			" + %g*(%s)"+
			" + %g*ln(1 + posts.likes_count + 2*posts.rebreeze_count + posts.comments_count)"+
			" + %g*exp(-EXTRACT(EPOCH FROM (now()-posts.created_at))/3600.0/%g)",
		wHash, wTheme, wAuthor, wFollow, followExpr, wPop, wRecency, recencyTauHours)
	sql += " ORDER BY (" + orderExpr + ") DESC, posts.created_at DESC"
	if len(followed) > 0 {
		args = append(args, followed)
	}
	sql += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	var posts []postModel
	if err := a.db.Raw(sql, args...).Scan(&posts).Error; err != nil {
		log.Printf("forYouFeed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{keyError: msgFetchPosts})
		return
	}
	ids := postIDs(posts)
	c.JSON(http.StatusOK, gin.H{
		keyData:  enrichPosts(a.db, posts, ids, callerID),
		keyPage:  page,
		keyLimit: limit,
		keyTotal: total,
	})
}

// followingIDs interroge le user-service (endpoint interne) pour la liste des IDs
// que viewerID suit, afin d'appliquer le bonus « abonné » dans le score Pour Toi.
// Même contrat best-effort que hiddenAuthors : en cas d'erreur réseau ou si
// USER_SERVICE_URL n'est pas configurée (dev/local), renvoie nil (pas de bonus).
func (a *app) followingIDs(viewerID string) []string {
	if a.userURL == "" || viewerID == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	url := fmt.Sprintf("%s/internal/users/%s/following-ids", a.userURL, neturl.PathEscape(viewerID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("X-Internal-Key", a.internalKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("followingIDs: %v", err)
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var body struct {
		Following []string `json:"following"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil
	}
	return body.Following
}
