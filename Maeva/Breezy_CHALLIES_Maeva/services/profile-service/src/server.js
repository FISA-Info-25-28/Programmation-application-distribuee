import app from "./app.js";
import { pool } from "./config/database.js";
import { initializeDatabase } from "./config/initDatabase.js";

const port = Number(
  process.env.PORT || 3002
);

let server;

async function startServer() {
  try {
    await initializeDatabase();

    server = app.listen(port, () => {
      console.log(
        `Profile Service démarré sur le port ${port}`
      );
    });
  } catch (error) {
    console.error(
      "Démarrage du Profile Service impossible :",
      error
    );

    process.exit(1);
  }
}

async function shutdown(signal) {
  console.log(
    `${signal} reçu : arrêt du Profile Service.`
  );

  if (!server) {
    await pool.end();
    process.exit(0);
  }

  server.close(async () => {
    await pool.end();
    process.exit(0);
  });
}

process.on(
  "SIGTERM",
  () => shutdown("SIGTERM")
);

process.on(
  "SIGINT",
  () => shutdown("SIGINT")
);

startServer();