// Le Pool n’est jamais réellement contacté ici : seules les règles
// de validation du contenu sont testées. Une URL fictive suffit pour
// satisfaire la vérification effectuée au chargement du module.
process.env.DATABASE_URL =
  process.env.DATABASE_URL ||
  "postgresql://test:test@localhost:5432/test";

import assert from "node:assert/strict";
import { test } from "node:test";

// Import dynamique : la variable d’environnement ci-dessus doit être
// définie avant le chargement de config/database.js, ce que les imports
// statiques (hoistés) ne garantissent pas.
const {
  MAX_CONTENT_LENGTH,
  normalizePostContent
} = await import("../src/services/postService.js");

test("un contenu valide est conservé après nettoyage", () => {
  const result = normalizePostContent(
    "  Premier message publié sur Breezy !  "
  );

  assert.equal(
    result,
    "Premier message publié sur Breezy !"
  );
});

test("un contenu vide est refusé (400)", () => {
  assert.throws(
    () => normalizePostContent("   "),
    (error) => error.statusCode === 400
  );
});

test("un contenu absent est refusé (400)", () => {
  assert.throws(
    () => normalizePostContent(undefined),
    (error) => error.statusCode === 400
  );
});

test("un contenu de plus de 280 caractères est refusé (400)", () => {
  const tooLong = "a".repeat(MAX_CONTENT_LENGTH + 1);

  assert.throws(
    () => normalizePostContent(tooLong),
    (error) => error.statusCode === 400
  );
});

test("un contenu de 280 caractères exactement est accepté", () => {
  const exact = "a".repeat(MAX_CONTENT_LENGTH);

  assert.equal(
    normalizePostContent(exact).length,
    MAX_CONTENT_LENGTH
  );
});
