// La validation est testée via registerUser() avec des données qui
// échouent AVANT tout appel en base. DATABASE_URL doit être définie
// pour que le module de base de données se charge sans erreur.
process.env.DATABASE_URL = "postgresql://test:test@localhost:5432/test";
process.env.JWT_SECRET = "secret-de-test";

import assert from "node:assert/strict";
import { test } from "node:test";

const { registerUser } = await import("../src/services/authService.js");

// ---- Validation de l'email ----

test("email sans arobase est refusé (400)", async () => {
  await assert.rejects(
    () => registerUser({ email: "pas-un-email", password: "motdepasse1", displayName: "Test" }),
    (err) => err.statusCode === 400
  );
});

test("email vide est refusé (400)", async () => {
  await assert.rejects(
    () => registerUser({ email: "", password: "motdepasse1", displayName: "Test" }),
    (err) => err.statusCode === 400
  );
});

test("email avec espaces seuls est refusé (400)", async () => {
  await assert.rejects(
    () => registerUser({ email: "   ", password: "motdepasse1", displayName: "Test" }),
    (err) => err.statusCode === 400
  );
});

// ---- Validation du mot de passe ----

test("mot de passe de 7 caractères est refusé (400)", async () => {
  await assert.rejects(
    () => registerUser({ email: "alice@test.com", password: "1234567", displayName: "Test" }),
    (err) => err.statusCode === 400
  );
});

test("mot de passe vide est refusé (400)", async () => {
  await assert.rejects(
    () => registerUser({ email: "alice@test.com", password: "", displayName: "Test" }),
    (err) => err.statusCode === 400
  );
});

test("mot de passe de 8 caractères passe la validation (erreur DB, pas 400)", async () => {
  await assert.rejects(
    () => registerUser({ email: "alice@test.com", password: "12345678", displayName: "Test" }),
    (err) => err.statusCode !== 400
  );
});

// ---- Validation du nom d'affichage ----

test("nom d'affichage vide est refusé (400)", async () => {
  await assert.rejects(
    () => registerUser({ email: "alice@test.com", password: "motdepasse1", displayName: "" }),
    (err) => err.statusCode === 400
  );
});

test("nom d'affichage composé d'espaces seuls est refusé (400)", async () => {
  await assert.rejects(
    () => registerUser({ email: "alice@test.com", password: "motdepasse1", displayName: "   " }),
    (err) => err.statusCode === 400
  );
});

test("nom d'affichage de 51 caractères est refusé (400)", async () => {
  await assert.rejects(
    () => registerUser({ email: "alice@test.com", password: "motdepasse1", displayName: "a".repeat(51) }),
    (err) => err.statusCode === 400
  );
});

test("nom d'affichage de 50 caractères passe la validation (erreur DB, pas 400)", async () => {
  await assert.rejects(
    () => registerUser({ email: "alice@test.com", password: "motdepasse1", displayName: "a".repeat(50) }),
    (err) => err.statusCode !== 400
  );
});
