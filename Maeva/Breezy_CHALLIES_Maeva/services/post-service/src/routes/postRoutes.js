import { Router } from "express";

import {
  listAllPosts,
  listMyPosts,
  publishPost
} from "../controllers/postController.js";

import { authenticate } from "../middlewares/authenticate.js";

const router = Router();

// Toutes les routes des publications exigent un JWT valide.

// Création d'une publication.
router.post(
  "/",
  authenticate,
  publishPost
);

// Fil d’actualité global (toutes les publications).
router.get(
  "/",
  authenticate,
  listAllPosts
);

// Publications de l'utilisateur connecté.
router.get(
  "/me",
  authenticate,
  listMyPosts
);

export default router;
