package models

import (
	"strings"
	"testing"
)

// tabler est implémenté par tous les modèles (méthode TableName de GORM).
type tabler interface{ TableName() string }

// TestTableNames vérifie que chaque modèle expose un nom de table non vide et
// unique. Les noms exacts sont par ailleurs exercés contre un vrai Postgres par
// les tests d'intégration (TRUNCATE), inutile de les redupliquer ici.
func TestTableNames(t *testing.T) {
	t.Parallel()
	all := []tabler{
		User{},
		Post{},
		PostTag{},
		Comment{},
		Like{},
		Follow{},
		Hashtag{},
		PostHashtag{},
		Media{},
		Mention{},
		Notification{},
		PrivateMessage{},
		Report{},
		Sanction{},
	}
	seen := make(map[string]bool, len(all))
	for _, m := range all {
		name := m.TableName()
		if strings.TrimSpace(name) == "" {
			t.Errorf("%T.TableName() is empty", m)
		}
		if seen[name] {
			t.Errorf("duplicate table name %q", name)
		}
		seen[name] = true
	}
	if len(seen) != len(all) {
		t.Errorf("got %d distinct table names, want %d", len(seen), len(all))
	}
}
