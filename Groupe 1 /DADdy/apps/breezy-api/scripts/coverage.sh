#!/usr/bin/env bash
# Calcule la couverture en excluant le code structurellement non testable listé
# dans .coverignore. Produit un profil filtré et affiche le total.
#
# Usage : scripts/coverage.sh [coverage.out]
#   - argument optionnel : chemin du profil brut (défaut: coverage.out)
#   - sortie : coverage.filtered.out + total imprimé sur stdout
set -euo pipefail

cd "$(dirname "$0")/.."

PROFILE="${1:-coverage.out}"
IGNORE_FILE=".coverignore"
FILTERED="coverage.filtered.out"

if [[ ! -f "$PROFILE" ]]; then
	echo "profil de couverture introuvable: $PROFILE" >&2
	exit 1
fi

# Motifs actifs : on ignore les lignes vides et les commentaires (#).
mapfile -t patterns < <(grep -vE '^[[:space:]]*(#|$)' "$IGNORE_FILE" || true)

if [[ ${#patterns[@]} -eq 0 ]]; then
	cp "$PROFILE" "$FILTERED"
else
	# Concatène les motifs en une seule alternation ERE.
	joined=$(IFS='|'; echo "${patterns[*]}")
	# On garde la ligne d'en-tête "mode:" + toutes les lignes non exclues.
	grep -vE "$joined" "$PROFILE" >"$FILTERED"
fi

echo "Couverture (hors code non testable, cf. .coverignore) :"
go tool cover -func="$FILTERED" | tail -1
