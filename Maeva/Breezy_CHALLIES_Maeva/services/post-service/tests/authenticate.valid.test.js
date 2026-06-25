import assert from "node:assert/strict";
import { test } from "node:test";
import jwt from "jsonwebtoken";

import { authenticate } from "../src/middlewares/authenticate.js";

const SECRET = "secret-de-test-unitaire";
process.env.JWT_SECRET = SECRET;

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

test("accepte un JWT valide et remplit req.user", () => {
  const token = jwt.sign(
    { email: "alice@test.com", role: "user" },
    SECRET,
    { subject: "uuid-alice", expiresIn: "1h" }
  );
  const req = { headers: { authorization: `Bearer ${token}` } };
  const res = buildResponse();
  let nextCalled = false;

  authenticate(req, res, () => { nextCalled = true; });

  assert.equal(nextCalled, true);
  assert.equal(res.statusCode, undefined);
  assert.equal(req.user.sub, "uuid-alice");
  assert.equal(req.user.email, "alice@test.com");
  assert.equal(req.user.role, "user");
});

test("refuse un JWT signé avec un mauvais secret (401)", () => {
  const token = jwt.sign(
    { email: "hack@test.com", role: "admin" },
    "mauvais-secret",
    { subject: "uuid-pirate" }
  );
  const req = { headers: { authorization: `Bearer ${token}` } };
  const res = buildResponse();
  let nextCalled = false;

  authenticate(req, res, () => { nextCalled = true; });

  assert.equal(res.statusCode, 401);
  assert.equal(nextCalled, false);
});

test("refuse un JWT expiré (401)", () => {
  const token = jwt.sign(
    { email: "bob@test.com", role: "user" },
    SECRET,
    { subject: "uuid-bob", expiresIn: "-1s" }
  );
  const req = { headers: { authorization: `Bearer ${token}` } };
  const res = buildResponse();
  let nextCalled = false;

  authenticate(req, res, () => { nextCalled = true; });

  assert.equal(res.statusCode, 401);
  assert.equal(nextCalled, false);
});
