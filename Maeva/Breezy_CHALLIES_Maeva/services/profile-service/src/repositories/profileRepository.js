import { pool } from "../config/database.js";

/**
 * Crée un profil lors de l'inscription d'un utilisateur.
 *
 * Cette fonction est utilisée par la route interne appelée
 * depuis l'Auth Service.
 */
export async function createProfile({
  userId,
  displayName
}) {
  const result = await pool.query(
    `
      INSERT INTO profiles (
        user_id,
        display_name
      )
      VALUES ($1, $2)
      RETURNING
        user_id AS "userId",
        display_name AS "displayName",
        bio,
        avatar_url AS "avatarUrl",
        created_at AS "createdAt",
        updated_at AS "updatedAt"
    `,
    [
      userId,
      displayName
    ]
  );

  return result.rows[0];
}

/**
 * Recherche un profil à partir de son identifiant utilisateur.
 *
 * @param {string} userId Identifiant provenant du JWT.
 * @returns {Promise<object|null>} Profil trouvé ou null.
 */
export async function findProfileByUserId(userId) {
  const result = await pool.query(
    `
      SELECT
        user_id AS "userId",
        display_name AS "displayName",
        bio,
        avatar_url AS "avatarUrl",
        created_at AS "createdAt",
        updated_at AS "updatedAt"
      FROM profiles
      WHERE user_id = $1
    `,
    [
      userId
    ]
  );

  // Retourne null lorsqu'aucune ligne ne correspond.
  return result.rows[0] ?? null;
}

/**
 * Récupère les informations publiques de plusieurs profils.
 *
 * Utilisé par le frontend pour afficher le nom et l'avatar des auteurs
 * du fil d'actualité, à partir des author_id fournis par le Post Service.
 *
 * @param {string[]} userIds Liste d'identifiants utilisateur.
 * @returns {Promise<object[]>} Profils publics correspondants.
 */
export async function findPublicProfilesByUserIds(userIds) {
  const result = await pool.query(
    `
      SELECT
        user_id AS "userId",
        display_name AS "displayName",
        avatar_url AS "avatarUrl"
      FROM profiles
      WHERE user_id = ANY($1::uuid[])
    `,
    [userIds]
  );

  return result.rows;
}

/**
 * Met à jour les informations publiques d'un profil.
 *
 * Les trois valeurs finales sont fournies par la couche service.
 * Cela permet de gérer correctement une requête PATCH partielle.
 */
export async function updateProfileByUserId({
  userId,
  displayName,
  bio,
  avatarUrl
}) {
  const result = await pool.query(
    `
      UPDATE profiles
      SET
        display_name = $2,
        bio = $3,
        avatar_url = $4,
        updated_at = NOW()
      WHERE user_id = $1
      RETURNING
        user_id AS "userId",
        display_name AS "displayName",
        bio,
        avatar_url AS "avatarUrl",
        created_at AS "createdAt",
        updated_at AS "updatedAt"
    `,
    [
      userId,
      displayName,
      bio,
      avatarUrl
    ]
  );

  // Retourne le profil après modification.
  return result.rows[0] ?? null;
}