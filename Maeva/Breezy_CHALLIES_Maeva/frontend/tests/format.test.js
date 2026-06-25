import assert from "node:assert/strict";
import { test } from "node:test";

import { formatDate, handleFromName } from "../lib/format.js";

// ---- formatDate ----

test("formatDate retourne une chaîne non vide pour une date valide", () => {
  const result = formatDate("2024-01-15T10:30:00Z");
  assert.equal(typeof result, "string");
  assert.ok(result.length > 0);
});

test("formatDate retourne une chaîne vide pour une date invalide", () => {
  assert.equal(formatDate("pas-une-date"), "");
});

test("formatDate retourne une chaîne vide pour une chaîne vide", () => {
  assert.equal(formatDate(""), "");
});

test("formatDate retourne une chaîne non vide pour null (new Date(null) = epoch)", () => {
  const result = formatDate(null);
  assert.equal(typeof result, "string");
  assert.ok(result.length > 0);
});

test("formatDate retourne une chaîne vide pour undefined", () => {
  assert.equal(formatDate(undefined), "");
});

test("formatDate retourne une chaîne vide pour NaN", () => {
  assert.equal(formatDate("NaN"), "");
});

// ---- handleFromName ----

test("handleFromName retourne @utilisateur pour une valeur vide", () => {
  assert.equal(handleFromName(""), "@utilisateur");
});

test("handleFromName retourne @utilisateur pour null", () => {
  assert.equal(handleFromName(null), "@utilisateur");
});

test("handleFromName retourne @utilisateur pour undefined", () => {
  assert.equal(handleFromName(undefined), "@utilisateur");
});

test("handleFromName produit un slug en minuscules", () => {
  assert.equal(handleFromName("Alice"), "@alice");
  assert.equal(handleFromName("BOB"), "@bob");
});

test("handleFromName retire les accents", () => {
  assert.equal(handleFromName("Élodie"), "@elodie");
  assert.equal(handleFromName("François"), "@francois");
});

test("handleFromName supprime les espaces", () => {
  assert.equal(handleFromName("Marie Claire"), "@marieclaire");
});

test("handleFromName supprime les tirets", () => {
  assert.equal(handleFromName("Jean-Luc"), "@jeanluc");
});

test("handleFromName conserve les chiffres", () => {
  assert.equal(handleFromName("User123"), "@user123");
});

test("handleFromName avec que des caractères spéciaux → @utilisateur", () => {
  assert.equal(handleFromName("!!!"), "@utilisateur");
  assert.equal(handleFromName("---"), "@utilisateur");
});

test("handleFromName avec que des espaces → @utilisateur", () => {
  assert.equal(handleFromName("   "), "@utilisateur");
});
