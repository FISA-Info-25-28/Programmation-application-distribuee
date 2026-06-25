import { pool } from "../config/database.js";

/**
 * Recherche un utilisateur par son email (déjà normalisé en minuscules).
 *
 * Le hash du mot de passe est renvoyé ici, car cette fonction sert à la
 * connexion. Retourne null lorsqu'aucun compte ne correspond.
 */
export async function findUserByEmail(email) {
  const result = await pool.query(
    `
      SELECT
        id,
        email,
        password_hash AS "passwordHash",
        role,
        created_at AS "createdAt"
      FROM users
      WHERE email = $1
    `,
    [email]
  );

  return result.rows[0] ?? null;
}

/**
 * Recherche un utilisateur par son identifiant.
 *
 * Le hash du mot de passe n'est volontairement pas sélectionné : cette
 * fonction sert aux lectures de profil (route /me).
 */
export async function findUserById(id) {
  const result = await pool.query(
    `
      SELECT
        id,
        email,
        role,
        created_at AS "createdAt"
      FROM users
      WHERE id = $1
    `,
    [id]
  );

  return result.rows[0] ?? null;
}

/**
 * Insère un nouvel utilisateur et renvoie ses informations publiques.
 *
 * Une violation de la contrainte d'unicité (email déjà pris) remonte sous
 * la forme d'une erreur PostgreSQL de code 23505, traitée par le service.
 */
export async function createUser({
  id,
  email,
  passwordHash,
  role = "user"
}) {
  const result = await pool.query(
    `
      INSERT INTO users (
        id,
        email,
        password_hash,
        role
      )
      VALUES ($1, $2, $3, $4)
      RETURNING
        id,
        email,
        role,
        created_at AS "createdAt"
    `,
    [id, email, passwordHash, role]
  );

  return result.rows[0];
}

/**
 * Supprime un utilisateur par son identifiant.
 *
 * Utilisé en compensation lorsque la création du profil échoue après la
 * création du compte (voir authService.registerUser).
 */
export async function deleteUserById(id) {
  await pool.query(
    `
      DELETE FROM users
      WHERE id = $1
    `,
    [id]
  );
}