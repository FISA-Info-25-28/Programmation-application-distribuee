import { Router } from "express";

import { createInternalProfile } from "../controllers/internalProfileController.js";
import { verifyInternalApiKey } from "../middlewares/internalApiKey.js";

const router = Router();

/*
 * Routes internes (service à service), montées sous /internal et protégées
 * par clé API. Utilisées par l'Auth Service lors de l'inscription, jamais
 * exposées au navigateur via la gateway.
 */
router.post(
  "/profiles",
  verifyInternalApiKey,
  createInternalProfile
);

export default router;