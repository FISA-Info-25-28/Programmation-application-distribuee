import pg from "pg";

const { Pool } = pg;

// Chaîne de connexion injectée par l'environnement (docker-compose / .env).
const connectionString = process.env.DATABASE_URL;

// Échec immédiat au chargement plutôt qu'une erreur obscure à la 1re requête.
if (!connectionString) {
  throw new Error("La variable DATABASE_URL est obligatoire.");
}

// Pool partagé par tout le service : réutilise les connexions PostgreSQL.
export const pool = new Pool({
  connectionString
});