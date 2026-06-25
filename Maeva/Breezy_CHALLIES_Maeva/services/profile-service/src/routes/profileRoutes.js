import { Router } from "express";

import {
  getMyProfile,
  getProfilesBatch,
  updateMyProfile
} from "../controllers/profileController.js";

import { authenticate } from "../middlewares/authenticate.js";

const router = Router();

/**
 * Toutes les routes de ce fichier concernent
 * le profil de l'utilisateur connecté.
 */

// Résolution par lot des profils publics (?ids=uuid1,uuid2,...).
// Déclarée avant /me : routes littérales distinctes, sans ambiguïté.
router.get(
  "/batch",
  authenticate,
  getProfilesBatch
);

// Consultation du profil courant.
router.get(
  "/me",
  authenticate,
  getMyProfile
);

// Modification partielle du profil courant.
router.patch(
  "/me",
  authenticate,
  updateMyProfile
);

export default router;