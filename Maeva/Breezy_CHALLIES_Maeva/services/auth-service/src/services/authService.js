import { randomUUID } from "node:crypto";
import bcrypt from "bcryptjs";
import jwt from "jsonwebtoken";

import {
  createUser,
  deleteUserById,
  findUserByEmail,
  findUserById
} from "../repositories/userRepository.js";

import { createProfile } from "./profileClient.js";
import { HttpError } from "../utils/HttpError.js";

const EMAIL_PATTERN =
  /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

// Normalise un email pour le comparer et le stocker de façon cohérente.
function normalizeEmail(email) {
  return email.trim().toLowerCase();
}

/*
 * Valide les données d'inscription et lève une HttpError 400 à la première
 * règle non respectée (format de l'email, longueur du mot de passe et du
 * nom d'affichage).
 */
function validateRegistration({
  email,
  password,
  displayName
}) {
  if (
    typeof email !== "string" ||
    !EMAIL_PATTERN.test(email.trim())
  ) {
    throw new HttpError(
      400,
      "L'adresse email est invalide."
    );
  }

  if (
    typeof password !== "string" ||
    password.length < 8
  ) {
    throw new HttpError(
      400,
      "Le mot de passe doit contenir au moins 8 caractères."
    );
  }

  if (
    typeof displayName !== "string" ||
    displayName.trim().length === 0
  ) {
    throw new HttpError(
      400,
      "Le nom d'affichage est obligatoire."
    );
  }

  if (displayName.trim().length > 50) {
    throw new HttpError(
      400,
      "Le nom d'affichage ne doit pas dépasser 50 caractères."
    );
  }
}

/*
 * Génère un JWT signé pour l'utilisateur. Le secret est obligatoire ;
 * l'identifiant utilisateur est placé dans le `subject` (claim `sub`).
 */
function generateAccessToken(user) {
  const jwtSecret = process.env.JWT_SECRET;
  const jwtExpiresIn =
    process.env.JWT_EXPIRES_IN || "1h";

  if (!jwtSecret) {
    throw new Error(
      "La variable JWT_SECRET est obligatoire."
    );
  }

  return jwt.sign(
    {
      email: user.email,
      role: user.role
    },
    jwtSecret,
    {
      subject: user.id,
      expiresIn: jwtExpiresIn
    }
  );
}

/*
 * Inscrit un nouvel utilisateur : valide les données, hache le mot de
 * passe, crée le compte puis crée le profil associé dans le Profile
 * Service. Si la création du profil échoue, le compte est supprimé en
 * compensation afin de ne pas laisser un utilisateur sans profil.
 */
export async function registerUser({
  email,
  password,
  displayName
}) {
  validateRegistration({
    email,
    password,
    displayName
  });

  const normalizedEmail = normalizeEmail(email);
  const normalizedDisplayName = displayName.trim();

  // Pré-vérification : évite un appel inutile lorsque l'email est déjà pris.
  const existingUser =
    await findUserByEmail(normalizedEmail);

  if (existingUser) {
    throw new HttpError(
      409,
      "Un compte existe déjà avec cette adresse email."
    );
  }

  const userId = randomUUID();

  // bcrypt avec un coût de 12 : jamais le mot de passe en clair en base.
  const passwordHash = await bcrypt.hash(
    password,
    12
  );

  let createdUser;

  try {
    createdUser = await createUser({
      id: userId,
      email: normalizedEmail,
      passwordHash,
      role: "user"
    });
  } catch (error) {
    // 23505 = violation d'unicité : un compte a été créé entre-temps.
    if (error.code === "23505") {
      throw new HttpError(
        409,
        "Un compte existe déjà avec cette adresse email."
      );
    }

    throw error;
  }

  try {
    await createProfile({
      userId: createdUser.id,
      displayName: normalizedDisplayName
    });
  } catch (error) {
    /*
     * Compensation minimale :
     * si le profil ne peut pas être créé,
     * le compte créé dans auth_db est supprimé.
     */
    await deleteUserById(createdUser.id);

    console.error(
      "Création du profil impossible :",
      error.message
    );

    throw new HttpError(
      502,
      "La création du profil a échoué. Le compte n'a pas été conservé."
    );
  }

  return createdUser;
}

/*
 * Authentifie un utilisateur et renvoie un jeton d'accès. Les messages
 * d'erreur restent volontairement génériques (« Identifiants incorrects »)
 * pour ne pas révéler si l'email existe.
 */
export async function loginUser({
  email,
  password
}) {
  if (
    typeof email !== "string" ||
    typeof password !== "string"
  ) {
    throw new HttpError(
      400,
      "L'adresse email et le mot de passe sont obligatoires."
    );
  }

  const normalizedEmail = normalizeEmail(email);

  const user =
    await findUserByEmail(normalizedEmail);

  if (!user) {
    throw new HttpError(
      401,
      "Identifiants incorrects."
    );
  }

  const passwordMatches = await bcrypt.compare(
    password,
    user.passwordHash
  );

  if (!passwordMatches) {
    throw new HttpError(
      401,
      "Identifiants incorrects."
    );
  }

  const accessToken = generateAccessToken(user);

  return {
    accessToken,

    tokenType: "Bearer",

    expiresIn:
      process.env.JWT_EXPIRES_IN || "1h",

    user: {
      id: user.id,
      email: user.email,
      role: user.role
    }
  };
}

/*
 * Charge l'utilisateur courant à partir de son identifiant (issu du JWT).
 * Lève une 404 si le compte a disparu depuis l'émission du jeton.
 */
export async function getAuthenticatedUser(userId) {
  const user = await findUserById(userId);

  if (!user) {
    throw new HttpError(
      404,
      "Utilisateur introuvable."
    );
  }

  return user;
}