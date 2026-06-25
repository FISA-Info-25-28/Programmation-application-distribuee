import { pool } from "./database.js";

/*
 * Crée la table posts et son index si nécessaire. Idempotent : exécuté à
 * chaque démarrage, il ne fait rien lorsque le schéma est déjà en place.
 * L'index (author_id, created_at DESC) accélère la liste des posts d'un
 * auteur, triée du plus récent au plus ancien.
 */
export async function initializeDatabase() {
  await pool.query(`
    CREATE TABLE IF NOT EXISTS posts (
      id UUID PRIMARY KEY,
      author_id UUID NOT NULL,
      content VARCHAR(280) NOT NULL,
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

      CONSTRAINT content_not_empty
        CHECK (CHAR_LENGTH(TRIM(content)) > 0)
    );
  `);

  await pool.query(`
    CREATE INDEX IF NOT EXISTS posts_author_created_idx
      ON posts (author_id, created_at DESC);
  `);

  console.log("Table post.posts initialisée.");
}
