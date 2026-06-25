package main

import (
	"time"

	"gorm.io/gorm"
)

// hashtagNeighborModel matérialise la PROXIMITÉ entre hashtags, calculée par
// co-engagement : deux hashtags sont proches s'ils sont aimés / rebreezés /
// commentés par les mêmes utilisateurs. On n'a pas besoin de savoir que « Messi
// joue au foot » : on l'observe (les fans de sport engagent #messi ET #sport).
// Chaque hashtag est une « entité » (un nœud) ; #messi et #foot restent
// distincts, simplement reliés par une arête de poids fort. Lignes orientées
// (tag → neighbor) pour des requêtes simples. C'est la matière première du
// regroupement en thèmes émergents (cf. clusters.go).
type hashtagNeighborModel struct {
	Tag       string    `gorm:"primaryKey;size:100"`
	Neighbor  string    `gorm:"primaryKey;size:100"`
	Score     float64   `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (hashtagNeighborModel) TableName() string { return "hashtag_neighbors" }

const neighborMinCommonUsers = 2 // au moins N audiences communes pour relier 2 tags

// refreshHashtagNeighbors recalcule toute la matrice de proximité depuis
// user_affinity (kind=hashtag). Score = Σ_users min(affinité(u,a), affinité(u,b))
// = recouvrement d'audience entre deux hashtags. Best-effort, idempotent
// (remplace la table dans une transaction courte). Appelé périodiquement.
func (a *app) refreshHashtagNeighbors() {
	type pair struct {
		Tag      string
		Neighbor string
		Score    float64
	}
	var pairs []pair
	a.db.Raw(`
		SELECT a.ref AS tag, b.ref AS neighbor, SUM(LEAST(a.weight, b.weight)) AS score
		FROM user_affinity a
		JOIN user_affinity b
		  ON a.user_id = b.user_id AND b.kind = ? AND a.ref <> b.ref
		WHERE a.kind = ?
		GROUP BY a.ref, b.ref
		HAVING COUNT(*) >= ?`,
		affinityKindHashtag, affinityKindHashtag, neighborMinCommonUsers).Scan(&pairs)

	now := time.Now().UTC()
	rows := make([]hashtagNeighborModel, 0, len(pairs))
	for _, p := range pairs {
		if p.Score <= 0 {
			continue
		}
		rows = append(rows, hashtagNeighborModel{Tag: p.Tag, Neighbor: p.Neighbor, Score: p.Score, UpdatedAt: now})
	}

	_ = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM hashtag_neighbors").Error; err != nil {
			return err
		}
		if len(rows) > 0 {
			return tx.CreateInBatches(rows, 500).Error
		}
		return nil
	})
}
