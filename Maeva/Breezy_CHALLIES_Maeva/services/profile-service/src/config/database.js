import pg from "pg";

const { Pool } = pg;

const connectionString = process.env.DATABASE_URL;

if (!connectionString) {
  throw new Error("La variable DATABASE_URL est obligatoire.");
}

export const pool = new Pool({
  connectionString
});