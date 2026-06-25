package main

import (
	"math"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// hashtagClusterModel relie un hashtag au cluster émergent auquel il appartient
// (= son thème). Le cluster est étiqueté par son hashtag le plus central (degré
// le plus fort). Reconstruit périodiquement à partir du graphe de co-engagement
// (hashtag_neighbors) : aucune liste de thèmes n'est codée en dur, tout émerge
// des interactions.
type hashtagClusterModel struct {
	Tag       string    `gorm:"primaryKey;size:100"`
	Cluster   string    `gorm:"size:100;not null;index"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (hashtagClusterModel) TableName() string { return "hashtag_clusters" }

// clusterTopK borne le voisinage considéré par hashtag lors du regroupement.
const clusterTopK = 4

// refreshHashtagClusters détecte les communautés de hashtags sur le graphe de
// co-engagement (top-K cosine + label propagation pondérée), puis remplace la
// table hashtag_clusters. Chaque cluster d'au moins deux membres est étiqueté par
// son hashtag de plus fort degré. Best-effort.
func (a *app) refreshHashtagClusters() {
	deg, adj := a.loadHashtagGraph()
	if len(adj) == 0 {
		return
	}
	cosw := topKCosineNeighbors(deg, adj)
	tags := sortedKeys(cosw)
	label := propagateLabels(tags, cosw)
	rows := buildClusterRows(tags, label, deg)

	_ = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM hashtag_clusters").Error; err != nil {
			return err
		}
		if len(rows) > 0 {
			return tx.CreateInBatches(rows, 500).Error
		}
		return nil
	})
}

// loadHashtagGraph charge le graphe de co-engagement : degré de chaque hashtag
// (somme de ses arêtes) et adjacence pondérée.
func (a *app) loadHashtagGraph() (map[string]float64, map[string]map[string]float64) {
	type edge struct {
		Tag      string
		Neighbor string
		Score    float64
	}
	var edges []edge
	a.db.Raw(`SELECT tag, neighbor, score FROM hashtag_neighbors`).Scan(&edges)

	deg := map[string]float64{}
	adj := map[string]map[string]float64{}
	for _, e := range edges {
		deg[e.Tag] += e.Score
		if adj[e.Tag] == nil {
			adj[e.Tag] = map[string]float64{}
		}
		adj[e.Tag][e.Neighbor] = e.Score
	}
	return deg, adj
}

// topKCosineNeighbors retient, par hashtag, ses top-K voisins par similarité
// cosine (score normalisé par les degrés → amortit les hubs), avec leur poids.
func topKCosineNeighbors(deg map[string]float64, adj map[string]map[string]float64) map[string]map[string]float64 {
	cosw := make(map[string]map[string]float64, len(adj))
	for t, neigh := range adj {
		type nb struct {
			tag string
			cos float64
		}
		list := make([]nb, 0, len(neigh))
		for n, s := range neigh {
			list = append(list, nb{n, s / math.Sqrt(deg[t]*deg[n])})
		}
		sort.Slice(list, func(i, j int) bool {
			if list[i].cos != list[j].cos {
				return list[i].cos > list[j].cos
			}
			return list[i].tag < list[j].tag
		})
		m := map[string]float64{}
		for i := 0; i < len(list) && i < clusterTopK; i++ {
			m[list[i].tag] = list[i].cos
		}
		cosw[t] = m
	}
	return cosw
}

func sortedKeys(m map[string]map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// propagateLabels applique une label propagation pondérée (asynchrone,
// stable) : chaque hashtag adopte le label dominant chez ses voisins top-K,
// jusqu'à convergence. Les communautés denses se renforcent sans qu'un simple
// pont ne fusionne tout.
func propagateLabels(tags []string, cosw map[string]map[string]float64) map[string]string {
	label := make(map[string]string, len(tags))
	for _, t := range tags {
		label[t] = t
	}
	for iter := 0; iter < 20; iter++ {
		if !labelPass(tags, cosw, label) {
			break
		}
	}
	return label
}

// labelPass effectue une passe de propagation et indique si un label a changé.
func labelPass(tags []string, cosw map[string]map[string]float64, label map[string]string) bool {
	changed := false
	for _, t := range tags {
		best := bestNeighborLabel(cosw[t], label)
		if best != "" && best != label[t] {
			label[t] = best
			changed = true
		}
	}
	return changed
}

// bestNeighborLabel renvoie le label le plus voté (pondéré cosine) parmi les
// voisins, avec un départage stable (plus petit label).
func bestNeighborLabel(neighbors map[string]float64, label map[string]string) string {
	votes := map[string]float64{}
	for n, w := range neighbors {
		votes[label[n]] += w
	}
	best := ""
	bestV := -1.0
	for lab, v := range votes {
		if v > bestV || (v == bestV && lab < best) {
			bestV, best = v, lab
		}
	}
	return best
}

// buildClusterRows regroupe les hashtags par label final et produit les lignes à
// persister, en ignorant les singletons (signal insuffisant).
func buildClusterRows(tags []string, label map[string]string, deg map[string]float64) []hashtagClusterModel {
	members := map[string][]string{}
	for _, t := range tags {
		members[label[t]] = append(members[label[t]], t)
	}
	now := time.Now().UTC()
	rows := make([]hashtagClusterModel, 0)
	for _, ms := range members {
		if len(ms) < 2 {
			continue
		}
		lab := clusterLabel(ms, deg)
		for _, m := range ms {
			rows = append(rows, hashtagClusterModel{Tag: m, Cluster: lab, UpdatedAt: now})
		}
	}
	return rows
}

// clusterLabel choisit l'étiquette d'un cluster : son membre de plus fort degré
// (départage stable par ordre alphabétique).
func clusterLabel(members []string, deg map[string]float64) string {
	lab := members[0]
	for _, m := range members {
		if deg[m] > deg[lab] || (deg[m] == deg[lab] && m < lab) {
			lab = m
		}
	}
	return lab
}

// refreshThemeAffinity (re)dérive l'affinité-thème de chaque utilisateur à partir
// de son affinité-hashtag et des clusters : weight(user, cluster) = Σ des
// affinités-hashtag du user pour les hashtags de ce cluster. Le thème n'est donc
// jamais un signal stocké au moment de l'engagement (où le post n'a pas encore de
// thème) : il découle des hashtags réellement engagés + des clusters émergents.
// Recalculé à chaque passe (remplace les lignes kind=theme). Best-effort.
func (a *app) refreshThemeAffinity() {
	_ = a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM user_affinity WHERE kind = ?`, affinityKindTheme).Error; err != nil {
			return err
		}
		return tx.Exec(`
			INSERT INTO user_affinity (user_id, kind, ref, weight, updated_at)
			SELECT ua.user_id, ?, hc.cluster, SUM(ua.weight), now()
			FROM user_affinity ua
			JOIN hashtag_clusters hc ON hc.tag = ua.ref
			WHERE ua.kind = ?
			GROUP BY ua.user_id, hc.cluster`,
			affinityKindTheme, affinityKindHashtag).Error
	})
}

// clusterThemeForTags renvoie le thème (label de cluster) majoritaire parmi les
// hashtags fournis, ou nil si aucun n'appartient encore à un cluster.
func clusterThemeForTags(db *gorm.DB, tags []string) *string {
	if len(tags) == 0 {
		return nil
	}
	lowered := make([]string, 0, len(tags))
	for _, t := range tags {
		lowered = append(lowered, strings.ToLower(t))
	}
	var row struct {
		Cluster string
		N       int
	}
	if err := db.Raw(`
		SELECT cluster, COUNT(*) AS n
		FROM hashtag_clusters
		WHERE tag IN ?
		GROUP BY cluster
		ORDER BY n DESC, cluster ASC
		LIMIT 1`, lowered).Scan(&row).Error; err != nil || row.Cluster == "" {
		return nil
	}
	return &row.Cluster
}
