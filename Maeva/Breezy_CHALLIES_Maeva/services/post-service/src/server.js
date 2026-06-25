import app from "./app.js";
import { pool } from "./config/database.js";
import { initializeDatabase } from "./config/initDatabase.js";

const port = Number(
  process.env.PORT || 3003
);

let server;

/*
 * Point d'entrée du service : initialise le schéma de la base puis met le
 * serveur HTTP en écoute. Tout échec au démarrage interrompt le processus.
 */
async function startServer() {
  try {
    // Crée les tables si elles n'existent pas encore.
    await initializeDatabase();

    server = app.listen(port, () => {
      console.log(
        `Post Service démarré sur le port ${port}`
      );
    });
  } catch (error) {
    console.error(
      "Démarrage du Post Service impossible :",
      error
    );

    process.exit(1);
  }
}

/*
 * Arrêt propre : ferme le serveur HTTP puis le pool de connexions à la
 * base avant de quitter, afin de ne pas laisser de connexions ouvertes.
 */
async function shutdown(signal) {
  console.log(
    `${signal} reçu : arrêt du Post Service.`
  );

  // Si le serveur n'a jamais démarré, on ferme directement le pool.
  if (!server) {
    await pool.end();
    process.exit(0);
  }

  server.close(async () => {
    await pool.end();
    process.exit(0);
  });
}

// SIGTERM est envoyé par Docker lors de l'arrêt du conteneur.
process.on(
  "SIGTERM",
  () => shutdown("SIGTERM")
);

// SIGINT correspond à un Ctrl+C en développement local.
process.on(
  "SIGINT",
  () => shutdown("SIGINT")
);

startServer();
