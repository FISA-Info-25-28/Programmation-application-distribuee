import { pool } from "./database.js";

export async function initializeDatabase() {
  await pool.query(`
    CREATE TABLE IF NOT EXISTS profiles (
      user_id UUID PRIMARY KEY,
      display_name VARCHAR(50) NOT NULL,
      bio VARCHAR(160),
      avatar_url VARCHAR(500),
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

      CONSTRAINT display_name_not_empty
        CHECK (CHAR_LENGTH(TRIM(display_name)) > 0)
    );
  `);

  console.log("Table profile.profiles initialisée.");
}