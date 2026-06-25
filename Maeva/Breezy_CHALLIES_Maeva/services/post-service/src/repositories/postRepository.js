import { pool } from "../config/database.js";

/**
 * Insère une publication et renvoie la ligne créée (avec ses dates).
 * Le contenu est supposé déjà validé et normalisé par la couche service.
 */
export async function insertPost({
  id,
  authorId,
  content
}) {
  const result = await pool.query(
    `
      INSERT INTO posts (
        id,
        author_id,
        content
      )
      VALUES ($1, $2, $3)
      RETURNING
        id,
        author_id AS "authorId",
        content,
        created_at AS "createdAt",
        updated_at AS "updatedAt"
    `,
    [id, authorId, content]
  );

  return result.rows[0];
}

/**
 * Renvoie les publications les plus récentes (fil global), limitées par
 * défaut à 100 pour borner la taille de la réponse.
 */
export async function findAllPosts(limit = 100) {
  const result = await pool.query(
    `
      SELECT
        id,
        author_id AS "authorId",
        content,
        created_at AS "createdAt",
        updated_at AS "updatedAt"
      FROM posts
      ORDER BY created_at DESC
      LIMIT $1
    `,
    [limit]
  );

  return result.rows;
}

/**
 * Renvoie toutes les publications d'un auteur, de la plus récente à la
 * plus ancienne (s'appuie sur l'index posts_author_created_idx).
 */
export async function findPostsByAuthorId(authorId) {
  const result = await pool.query(
    `
      SELECT
        id,
        author_id AS "authorId",
        content,
        created_at AS "createdAt",
        updated_at AS "updatedAt"
      FROM posts
      WHERE author_id = $1
      ORDER BY created_at DESC
    `,
    [authorId]
  );

  return result.rows;
}
