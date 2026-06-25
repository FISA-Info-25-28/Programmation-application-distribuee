package main

import (
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// userAffinityModel matérialise les préférences « organiques » d'un utilisateur :
// un poids accumulé par dimension (kind) et valeur (ref). Le poids grossit à
// chaque engagement (like/commentaire/rebreeze/bookmark/post rédigé) et décroît
// lentement avec le temps (cf. startAffinityDecay), si bien que les préférences
// suivent l'évolution réelle des centres d'intérêt. La PK composite
// (user_id, kind, ref) rend l'UPSERT idempotent.
type userAffinityModel struct {
	UserID    string    `gorm:"primaryKey;size:64"`
	Kind      string    `gorm:"primaryKey;size:16"`
	Ref       string    `gorm:"primaryKey;size:100"`
	Weight    float64   `gorm:"not null;default:0"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (userAffinityModel) TableName() string { return "user_affinity" }

const (
	affinityKindHashtag = "hashtag"
	affinityKindAuthor  = "author"
	affinityKindTheme   = "theme"
)

// Poids d'action : combien chaque type d'engagement pèse dans les préférences.
const (
	weightLike     = 1.0
	weightBookmark = 2.0
	weightComment  = 2.0
	weightRebreeze = 3.0
	weightAuthored = 3.0
)

// bumpAffinity ajuste (delta peut être négatif lors d'un un-like / un-rebreeze)
// le poids d'affinité de userID sur les trois dimensions d'un post : ses
// hashtags, son auteur et son thème. L'affinité est un signal best-effort : une
// erreur ici ne doit jamais faire échouer l'action utilisateur, donc on ignore
// les erreurs. La dimension auteur est ignorée quand l'auteur est l'utilisateur
// lui-même (on ne « s'aime » pas soi-même). L'affinité-THÈME n'est PAS bumpée
// ici : elle est dérivée a posteriori des affinités-hashtag + des clusters
// émergents (cf. refreshThemeAffinity), pour que le thème reste 100% émergent.
func bumpAffinity(db *gorm.DB, userID string, post postModel, tags []string, delta float64) {
	if userID == "" || delta == 0 {
		return
	}
	rows := make([]userAffinityModel, 0, len(tags)+1)
	for _, tag := range tags {
		rows = append(rows, userAffinityModel{
			UserID: userID, Kind: affinityKindHashtag, Ref: strings.ToLower(tag), Weight: delta,
		})
	}
	if post.AuthorID != "" && post.AuthorID != userID {
		rows = append(rows, userAffinityModel{
			UserID: userID, Kind: affinityKindAuthor, Ref: post.AuthorID, Weight: delta,
		})
	}
	if len(rows) == 0 {
		return
	}
	db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "kind"}, {Name: "ref"}},
		DoUpdates: clause.Assignments(map[string]any{
			"weight":     gorm.Expr("GREATEST(user_affinity.weight + ?, 0)", delta),
			"updated_at": time.Now().UTC(),
		}),
	}).Create(&rows)
}

// bumpAffinityForPost charge les hashtags du post puis applique le delta sur les
// trois dimensions d'affinité de userID. Utilisé par les handlers d'engagement
// (like/rebreeze/commentaire/bookmark).
func (a *app) bumpAffinityForPost(userID string, post postModel, delta float64) {
	tags := fetchHashtagsByPostIDs(a.db, []int64{post.ID})[post.ID]
	bumpAffinity(a.db, userID, post, tags, delta)
}

// startAffinityDecay lance une tâche de fond qui érode périodiquement tous les
// poids d'affinité (oubli progressif) et supprime les lignes devenues
// négligeables, pour garder des préférences « vivantes ». À appeler depuis main
// (jamais dans les tests).
func (a *app) startAffinityDecay(interval time.Duration, factor float64) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			a.db.Model(&userAffinityModel{}).
				Where("weight > 0").
				UpdateColumn("weight", gorm.Expr("weight * ?", factor))
			a.db.Where("weight < ?", 0.05).Delete(&userAffinityModel{})
		}
	}()
}
