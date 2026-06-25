package main

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// Déduction comportementale du thème : quand un breeze n'a aucun hashtag
// exploitable, on devine son thème d'après les centres d'intérêt des
// utilisateurs qui interagissent avec lui (un fan de plongée qui like → indice
// « plongee »), puis on confirme/corrige avec les interactions des autres.
const (
	themeInferMinEngagers = 3    // interactants distincts requis (vérification croisée)
	themeInferMinVoters   = 2    // interactants distincts soutenant le thème gagnant
	themeInferMinShare    = 0.10 // part minimale du thème chez les interactants
	themeInferLift        = 1.30 // sur-représentation vs moyenne globale (×1.3)
	themeInferHighShare   = 0.60 // …ou majorité écrasante, même sans sur-représentation
	themeInferBatch       = 300  // posts examinés par passe
)

// inferThemeFromEngagers déduit le thème d'un post à partir des affinités
// thématiques (user_affinity kind=theme) des utilisateurs qui interagissent avec
// lui — likes, rebreezes, commentaires. On choisit le thème le plus
// SUR-REPRÉSENTÉ chez ces interactants par rapport à la moyenne globale de la
// population (lift), ce qui neutralise le biais du thème globalement dominant
// (ex. « tech » présent partout) : un post qui attire disproportionnellement des
// fans de plongée est classé « plongee », même si ces mêmes users aiment aussi
// beaucoup de tech. Renvoie nil si le signal est insuffisant.
func inferThemeFromEngagers(db *gorm.DB, postID int64) *string {
	var engagers []string
	db.Raw(`
		SELECT user_id FROM likes WHERE post_id = ?
		UNION SELECT user_id FROM rebreezers WHERE post_id = ?
		UNION SELECT author_id FROM posts WHERE parent_id = ?`,
		postID, postID, postID).Scan(&engagers)
	if len(engagers) < themeInferMinEngagers {
		return nil
	}

	type tally struct {
		Theme  string
		Score  float64
		Voters int
	}
	var rows []tally
	db.Raw(`
		SELECT ref AS theme, SUM(weight) AS score, COUNT(DISTINCT user_id) AS voters
		FROM user_affinity
		WHERE kind = ? AND user_id IN ?
		GROUP BY ref`,
		affinityKindTheme, engagers).Scan(&rows)
	if len(rows) == 0 {
		return nil
	}

	// Baseline : distribution moyenne des thèmes sur toute la population.
	type grow struct {
		Theme string
		Score float64
	}
	var globalRows []grow
	db.Raw(`SELECT ref AS theme, SUM(weight) AS score FROM user_affinity WHERE kind = ? GROUP BY ref`,
		affinityKindTheme).Scan(&globalRows)
	globalByTheme := make(map[string]float64, len(globalRows))
	var globalTotal float64
	for _, g := range globalRows {
		globalByTheme[g.Theme] = g.Score
		globalTotal += g.Score
	}

	var postTotal float64
	for _, r := range rows {
		postTotal += r.Score
	}
	if postTotal <= 0 || globalTotal <= 0 {
		return nil
	}

	// On retient le thème au plus fort lift (part chez les interactants / part
	// globale), parmi ceux soutenus par assez d'interactants et pas anecdotiques.
	bestTheme := ""
	var bestLift, bestShare float64
	for _, r := range rows {
		if r.Voters < themeInferMinVoters {
			continue
		}
		share := r.Score / postTotal
		if share < themeInferMinShare {
			continue
		}
		gShare := globalByTheme[r.Theme] / globalTotal
		if gShare <= 0 {
			continue
		}
		lift := share / gShare
		if lift > bestLift {
			bestLift, bestTheme, bestShare = lift, r.Theme, share
		}
	}
	// Accepté si nettement sur-représenté, OU s'il écrase tout chez les
	// interactants (cas où peu de thèmes coexistent dans la population).
	if bestTheme == "" || (bestLift < themeInferLift && bestShare < themeInferHighShare) {
		return nil
	}
	return &bestTheme
}

// startThemeInference lance une passe périodique : (1) rafraîchit le graphe de
// proximité entre hashtags, (2) en déduit les clusters (= thèmes émergents),
// (3) étiquette les posts candidats. À appeler depuis main (jamais dans les tests).
func (a *app) startThemeInference(interval time.Duration) {
	go func() {
		time.Sleep(15 * time.Second) // laisse la stack et les migrations se stabiliser
		a.refreshHashtagNeighbors()
		a.refreshHashtagClusters()
		a.refreshThemeAffinity()
		a.runThemeInferencePass()
		ticker := time.NewTicker(interval)
		for range ticker.C {
			a.refreshHashtagNeighbors()
			a.refreshHashtagClusters()
			a.refreshThemeAffinity()
			a.runThemeInferencePass()
		}
	}()
}

// runThemeInferencePass étiquette un lot de posts candidats (posts racine sans
// thème ou déjà déduits → réévalués à chaque passe, les thèmes émergeant des
// données). Pour chacun, dans l'ordre : (1) cluster majoritaire de ses hashtags
// (le thème émerge du regroupement par co-engagement) ; (2) à défaut, déduction
// comportementale via les centres d'intérêt de ses interactants.
func (a *app) runThemeInferencePass() {
	var ids []int64
	a.db.Model(&postModel{}).
		Where("parent_id IS NULL AND (theme IS NULL OR theme_inferred = ?)", true).
		Limit(themeInferBatch).
		Pluck("id", &ids)
	if len(ids) == 0 {
		return
	}
	tagsByPost := fetchHashtagsByPostIDs(a.db, ids)
	updated := 0
	for _, id := range ids {
		theme := clusterThemeForTags(a.db, tagsByPost[id])
		if theme == nil {
			theme = inferThemeFromEngagers(a.db, id)
		}
		if theme == nil {
			continue
		}
		a.db.Model(&postModel{}).Where("id = ?", id).
			Updates(map[string]any{"theme": *theme, "theme_inferred": true})
		updated++
	}
	if updated > 0 {
		log.Printf("theme inference: %d post(s) étiqueté(s) (clusters de hashtags / comportement)", updated)
	}
}
