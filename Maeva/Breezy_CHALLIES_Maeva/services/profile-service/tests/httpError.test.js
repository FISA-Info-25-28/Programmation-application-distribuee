import assert from "node:assert/strict";
import { test } from "node:test";

import { HttpError } from "../src/utils/HttpError.js";

test("HttpError conserve le statusCode et le message", () => {
  const err = new HttpError(404, "Ressource introuvable.");
  assert.equal(err.statusCode, 404);
  assert.equal(err.message, "Ressource introuvable.");
});

test("HttpError est une instance d'Error", () => {
  const err = new HttpError(400, "Mauvaise requête.");
  assert.ok(err instanceof Error);
});

test("HttpError a la propriété name 'HttpError'", () => {
  const err = new HttpError(500, "Erreur serveur.");
  assert.equal(err.name, "HttpError");
});

test("HttpError génère une stack trace", () => {
  const err = new HttpError(403, "Accès interdit.");
  assert.ok(typeof err.stack === "string");
  assert.ok(err.stack.length > 0);
});

test("HttpError avec statusCode 200 ne lève pas d'exception à la construction", () => {
  assert.doesNotThrow(() => new HttpError(200, "OK."));
});
