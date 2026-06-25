import assert from "node:assert/strict";
import { test } from "node:test";

import { verifyInternalApiKey } from "../src/middlewares/internalApiKey.js";

function buildResponse() {
  return {
    statusCode: undefined,
    body: undefined,
    status(code) {
      this.statusCode = code;
      return this;
    },
    json(payload) {
      this.body = payload;
      return this;
    }
  };
}

test("accepte la clé correcte et appelle next()", () => {
  process.env.INTERNAL_API_KEY = "clé-secrète-interne";
  const req = { headers: { "x-internal-api-key": "clé-secrète-interne" } };
  const res = buildResponse();
  let nextCalled = false;

  verifyInternalApiKey(req, res, () => { nextCalled = true; });

  assert.equal(nextCalled, true);
  assert.equal(res.statusCode, undefined);
});

test("refuse une requête sans clé (401)", () => {
  process.env.INTERNAL_API_KEY = "clé-secrète-interne";
  const req = { headers: {} };
  const res = buildResponse();
  let nextCalled = false;

  verifyInternalApiKey(req, res, () => { nextCalled = true; });

  assert.equal(res.statusCode, 401);
  assert.equal(nextCalled, false);
});

test("refuse une clé incorrecte (401)", () => {
  process.env.INTERNAL_API_KEY = "clé-secrète-interne";
  const req = { headers: { "x-internal-api-key": "mauvaise-clé" } };
  const res = buildResponse();
  let nextCalled = false;

  verifyInternalApiKey(req, res, () => { nextCalled = true; });

  assert.equal(res.statusCode, 401);
  assert.equal(nextCalled, false);
});

test("refuse si INTERNAL_API_KEY n'est pas configurée (401)", () => {
  delete process.env.INTERNAL_API_KEY;
  const req = { headers: { "x-internal-api-key": "n-importe-quoi" } };
  const res = buildResponse();
  let nextCalled = false;

  verifyInternalApiKey(req, res, () => { nextCalled = true; });

  assert.equal(res.statusCode, 401);
  assert.equal(nextCalled, false);
});

test("la clé est sensible à la casse (401 si différente casse)", () => {
  process.env.INTERNAL_API_KEY = "CléSecrète";
  const req = { headers: { "x-internal-api-key": "clésecrete" } };
  const res = buildResponse();
  let nextCalled = false;

  verifyInternalApiKey(req, res, () => { nextCalled = true; });

  assert.equal(res.statusCode, 401);
  assert.equal(nextCalled, false);
});
