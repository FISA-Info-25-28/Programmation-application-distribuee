import assert from "node:assert/strict";
import { test } from "node:test";

import { authenticate } from "../src/middlewares/authenticate.js";

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

test("refuse l’accès sans en-tête Authorization (401)", () => {
  const req = { headers: {} };
  const res = buildResponse();
  let nextCalled = false;

  authenticate(req, res, () => {
    nextCalled = true;
  });

  assert.equal(res.statusCode, 401);
  assert.equal(nextCalled, false);
});

test("refuse un en-tête sans schéma Bearer (401)", () => {
  const req = {
    headers: { authorization: "Token abc" }
  };
  const res = buildResponse();
  let nextCalled = false;

  authenticate(req, res, () => {
    nextCalled = true;
  });

  assert.equal(res.statusCode, 401);
  assert.equal(nextCalled, false);
});

test("refuse un jeton Bearer invalide (401)", () => {
  process.env.JWT_SECRET =
    process.env.JWT_SECRET || "secret-de-test";

  const req = {
    headers: { authorization: "Bearer jeton-invalide" }
  };
  const res = buildResponse();
  let nextCalled = false;

  authenticate(req, res, () => {
    nextCalled = true;
  });

  assert.equal(res.statusCode, 401);
  assert.equal(nextCalled, false);
});
