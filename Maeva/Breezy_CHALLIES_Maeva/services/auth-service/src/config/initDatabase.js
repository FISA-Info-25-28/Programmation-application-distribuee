import { pool } from "./database.js";

/*
 * Crée la table users si elle n'existe pas. Idempotent : exécuté à chaque
 * démarrage, il ne fait rien lorsque le schéma est déjà en place.
 */
export async function initializeDatabase() {
  await pool.query(`
    CREATE TABLE IF NOT EXISTS users (
      id UUID PRIMARY KEY,
      email VARCHAR(255) UNIQUE NOT NULL,
      password_hash VARCHAR(255) NOT NULL,
      role VARCHAR(30) NOT NULL DEFAULT 'user',
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
  `);

  console.log("Table auth.users initialisée.");
}