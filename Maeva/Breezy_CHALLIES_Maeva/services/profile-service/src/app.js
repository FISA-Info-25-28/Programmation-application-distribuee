import express from "express";

import { pool } from "./config/database.js";

import internalProfileRoutes
  from "./routes/internalProfileRoutes.js";

import profileRoutes
  from "./routes/profileRoutes.js";

import { HttpError }
  from "./utils/HttpError.js";

const app = express();

// Masque l'information Express dans les en-têtes HTTP.
app.disable("x-powered-by");

// Autorise les corps JSON avec une taille maximale raisonnable.
app.use(express.json({
  limit: "20kb"
}));

/**
 * Route technique permettant de vérifier :
 * - le fonctionnement du service ;
 * - la connexion à profile_db.
 */
app.get("/health", async (_req, res) => {
  try {
    // Requête simple destinée au contrôle de la base.
    await pool.query("SELECT 1");

    return res.status(200).json({
      status: "ok",
      service: "profile-service",
      database: "connected"
    });
  } catch (error) {
    console.error(
      "Profile database health check failed:",
      error.message
    );

    return res.status(503).json({
      status: "error",
      service: "profile-service",
      database: "unavailable"
    });
  }
});

/**
 * Route interne utilisée uniquement par l'Auth Service
 * lors de la création automatique d'un profil.
 */
app.use(
  "/internal",
  internalProfileRoutes
);

/**
 * Routes publiques protégées par JWT.
 *
 * Chemins obtenus :
 * GET   /api/profile/me
 * PATCH /api/profile/me
 */
app.use(
  "/api/profile",
  profileRoutes
);

/**
 * Réponse utilisée lorsqu'aucune route ne correspond.
 */
app.use((_req, res) => {
  return res.status(404).json({
    message: "Route introuvable."
  });
});

/**
 * Middleware centralisé de gestion des erreurs.
 *
 * Il doit être déclaré après toutes les routes.
 */
app.use((error, _req, res, _next) => {
  // Erreur métier volontairement levée par le service.
  if (error instanceof HttpError) {
    return res.status(error.statusCode).json({
      message: error.message
    });
  }

  // L'erreur complète reste dans les logs du conteneur.
  console.error(
    "Erreur Profile Service :",
    error
  );

  // Le client reçoit un message générique.
  return res.status(500).json({
    message: "Une erreur interne est survenue."
  });
});

export default app;