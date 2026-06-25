import express from "express";

import { pool } from "./config/database.js";
import postRoutes from "./routes/postRoutes.js";
import { HttpError } from "./utils/HttpError.js";

const app = express();

// Masque l'information Express dans les en-têtes HTTP.
app.disable("x-powered-by");

// Autorise les corps JSON avec une taille maximale raisonnable.
app.use(express.json({
  limit: "20kb"
}));

/*
 * Route technique : vérifie le fonctionnement du service et la connexion
 * à post_db.
 */
app.get("/health", async (_req, res) => {
  try {
    // Requête simple destinée au contrôle de la base.
    await pool.query("SELECT 1");

    return res.status(200).json({
      status: "ok",
      service: "post-service",
      database: "connected"
    });
  } catch (error) {
    console.error(
      "Post database health check failed:",
      error.message
    );

    return res.status(503).json({
      status: "error",
      service: "post-service",
      database: "unavailable"
    });
  }
});

// Routes des publications, protégées par JWT (voir postRoutes).
app.use("/api/posts", postRoutes);

// Réponse utilisée lorsqu'aucune route ne correspond.
app.use((_req, res) => {
  return res.status(404).json({
    message: "Route introuvable."
  });
});

/*
 * Middleware centralisé de gestion des erreurs.
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
    "Erreur Post Service :",
    error
  );

  // Le client reçoit un message générique.
  return res.status(500).json({
    message: "Une erreur interne est survenue."
  });
});

export default app;
