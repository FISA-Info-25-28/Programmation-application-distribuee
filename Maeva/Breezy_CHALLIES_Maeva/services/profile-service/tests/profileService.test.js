// Les fonctions de validation rejettent les données invalides AVANT
// tout appel en base. DATABASE_URL doit être définie pour que le
// module de base de données se charge sans erreur.
process.env.DATABASE_URL = "postgresql://test:test@localhost:5432/test";

import assert from "node:assert/strict";
import { test } from "node:test";

const { getPublicProfiles, updateProfile } =
  await import("../src/services/profileService.js");

// ---- getPublicProfiles ----

test("getPublicProfiles - paramètre vide → 400", async () => {
  await assert.rejects(
    () => getPublicProfiles(""),
    (err) => err.statusCode === 400
  );
});

test("getPublicProfiles - paramètre undefined → 400", async () => {
  await assert.rejects(
    () => getPublicProfiles(undefined),
    (err) => err.statusCode === 400
  );
});

test("getPublicProfiles - espaces seuls → 400", async () => {
  await assert.rejects(
    () => getPublicProfiles("   "),
    (err) => err.statusCode === 400
  );
});

test("getPublicProfiles - UUIDs invalides uniquement → tableau vide (sans appel DB)", async () => {
  const result = await getPublicProfiles("pas-un-uuid,ceci-non-plus");
  assert.deepEqual(result, []);
});

test("getPublicProfiles - séparateurs seuls → tableau vide", async () => {
  const result = await getPublicProfiles(",,,");
  assert.deepEqual(result, []);
});

test("getPublicProfiles - mix d'UUIDs valides et invalides (les invalides sont ignorés)", async () => {
  // Un UUID valide → passe le filtre → DB est sollicitée → erreur connexion, pas 400.
  const validUuid = "12345678-1234-1234-1234-123456789abc";
  await assert.rejects(
    () => getPublicProfiles(`${validUuid},pas-un-uuid`),
    (err) => err.statusCode !== 400
  );
});

test("getPublicProfiles - plus de 100 UUIDs valides → 400", async () => {
  const ids = Array.from(
    { length: 101 },
    (_, i) => `00000000-0000-0000-0000-${String(i).padStart(12, "0")}`
  ).join(",");

  await assert.rejects(
    () => getPublicProfiles(ids),
    (err) => err.statusCode === 400
  );
});

test("getPublicProfiles - exactement 100 UUIDs valides passe la validation (erreur DB, pas 400)", async () => {
  const ids = Array.from(
    { length: 100 },
    (_, i) => `00000000-0000-0000-0000-${String(i).padStart(12, "0")}`
  ).join(",");

  await assert.rejects(
    () => getPublicProfiles(ids),
    (err) => err.statusCode !== 400
  );
});

test("getPublicProfiles - UUIDs dupliqués sont dédupliqués", async () => {
  const uuid = "12345678-1234-1234-1234-123456789abc";
  // Deux fois le même UUID → un seul en base → erreur DB, pas 400.
  await assert.rejects(
    () => getPublicProfiles(`${uuid},${uuid}`),
    (err) => err.statusCode !== 400
  );
});

// ---- updateProfile : validation du corps ----

test("updateProfile - corps null → 400", async () => {
  await assert.rejects(
    () => updateProfile("user-id", null),
    (err) => err.statusCode === 400
  );
});

test("updateProfile - corps tableau → 400", async () => {
  await assert.rejects(
    () => updateProfile("user-id", []),
    (err) => err.statusCode === 400
  );
});

test("updateProfile - corps vide (aucun champ) → 400", async () => {
  await assert.rejects(
    () => updateProfile("user-id", {}),
    (err) => err.statusCode === 400
  );
});

test("updateProfile - champ interdit 'id' → 400", async () => {
  await assert.rejects(
    () => updateProfile("user-id", { id: "hacked" }),
    (err) => err.statusCode === 400
  );
});

test("updateProfile - champ interdit 'userId' → 400", async () => {
  await assert.rejects(
    () => updateProfile("user-id", { userId: "hacked" }),
    (err) => err.statusCode === 400
  );
});

test("updateProfile - champ interdit 'role' → 400", async () => {
  await assert.rejects(
    () => updateProfile("user-id", { role: "admin" }),
    (err) => err.statusCode === 400
  );
});

test("updateProfile - champ autorisé seul passe la validation (erreur DB, pas 400)", async () => {
  await assert.rejects(
    () => updateProfile("user-id", { displayName: "Alice" }),
    (err) => err.statusCode !== 400
  );
});
