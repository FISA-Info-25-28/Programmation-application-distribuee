import { Router } from "express";

import {
  login,
  logout,
  me,
  register
} from "../controllers/authController.js";

import { authenticate } from "../middlewares/authenticate.js";

const router = Router();

// Routes publiques : création de compte et connexion.
router.post("/register", register);
router.post("/login", login);

// Routes protégées : nécessitent un JWT valide (middleware authenticate).
router.get(
  "/me",
  authenticate,
  me
);

router.post(
  "/logout",
  authenticate,
  logout
);

export default router;