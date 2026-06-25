package main

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"daddy/apps/breezy-api/internal/shared"
)

// requireInternalKey protège les routes internes service-à-service (clé comparée
// en temps constant). Si aucune clé n'est configurée, tout est refusé.
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

// suggestedAuthors répond à GET /internal/posts/suggested-authors?viewer=<id>&limit=N,
// appelé par le user-service pour ses suggestions d'abonnement « par thème ».
// Classe les auteurs par recouvrement entre l'affinité-thème du lecteur et les
// thèmes de leurs posts (theme inclut les thèmes déduits par comportement).
// Renvoie une liste d'IDs ordonnée — aucun thème, aucun profil n'est exposé.
func (a *app) suggestedAuthors(c *gin.Context) {
	viewer := strings.TrimSpace(c.Query("viewer"))
	limit := 30
	if n, err := strconv.Atoi(c.Query("limit")); err == nil && n > 0 && n <= 100 {
		limit = n
	}

	authors := []string{}
	if viewer != "" {
		// On classe par produit (affinité-thème du lecteur × PART thématique de
		// l'auteur). Utiliser la part normalisée (et non le comptage brut) évite que
		// les auteurs prolifiques sur le thème globalement dominant (tech) écrasent
		// les spécialistes du thème réellement aimé par le lecteur.
		a.db.Raw(`
			SELECT cnt.author_id
			FROM user_affinity va
			JOIN (
				SELECT author_id, theme,
				       COUNT(*)::float / SUM(COUNT(*)) OVER (PARTITION BY author_id) AS ashare
				FROM posts
				WHERE theme IS NOT NULL AND parent_id IS NULL
				GROUP BY author_id, theme
			) cnt ON cnt.theme = va.ref
			WHERE va.user_id = ? AND va.kind = ? AND cnt.author_id <> ?
			GROUP BY cnt.author_id
			ORDER BY SUM(va.weight * cnt.ashare) DESC
			LIMIT ?`,
			viewer, affinityKindTheme, viewer, limit).Scan(&authors)
	}
	c.JSON(http.StatusOK, gin.H{"authors": authors})
}
