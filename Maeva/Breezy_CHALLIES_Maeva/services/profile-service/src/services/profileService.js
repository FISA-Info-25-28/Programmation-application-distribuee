import {
  findProfileByUserId,
  findPublicProfilesByUserIds,
  updateProfileByUserId
} from "../repositories/profileRepository.js";

import { HttpError } from "../utils/HttpError.js";

// Format d'un identifiant UUID, pour filtrer les entrées invalides.
const UUID_PATTERN =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

// Garde-fou : nombre maximal d'identifiants résolus en une requête.
const MAX_BATCH_SIZE = 100;

/**
 * Champs que le client a le droit de transmettre.
 *
 * userId n'est volontairement pas présent :
 * l'identifiant provient uniquement du JWT.
 */
const ALLOWED_FIELDS = [
  "displayName",
  "bio",
  "avatarUrl"
];

/**
 * Vérifie si la valeur est une URL HTTP ou HTTPS valide.
 */
function isValidHttpUrl(value) {
  try {
    // Utilisation du parseur URL natif de Node.js.
    const parsedUrl = new URL(value);

    return (
      parsedUrl.protocol === "http:" ||
      parsedUrl.protocol === "https:"
    );
  } catch {
    // Une erreur signifie que la chaîne n'est pas une URL valide.
    return false;
  }
}

/**
 * Vérifie que le corps ne contient aucun champ interdit.
 */
function validateAllowedFields(payload) {
  const receivedFields = Object.keys(payload);

  // Recherche des propriétés non prévues dans l'API.
  const forbiddenFields = receivedFields.filter(
    (field) => !ALLOWED_FIELDS.includes(field)
  );

  if (forbiddenFields.length > 0) {
    throw new HttpError(
      400,
      `Champ(s) non autorisé(s) : ${forbiddenFields.join(", ")}.`
    );
  }

  // Une requête PATCH vide ne modifie rien.
  if (receivedFields.length === 0) {
    throw new HttpError(
      400,
      "Au moins un champ doit être fourni."
    );
  }
}

/**
 * Valide et normalise le nom affiché.
 */
function normalizeDisplayName(value) {
  if (typeof value !== "string") {
    throw new HttpError(
      400,
      "Le nom d'affichage doit être une chaîne de caractères."
    );
  }

  // Suppression des espaces inutiles.
  const normalizedValue = value.trim();

  if (normalizedValue.length === 0) {
    throw new HttpError(
      400,
      "Le nom d'affichage est obligatoire."
    );
  }

  if (normalizedValue.length > 50) {
    throw new HttpError(
      400,
      "Le nom d'affichage ne doit pas dépasser 50 caractères."
    );
  }

  return normalizedValue;
}

/**
 * Valide et normalise la biographie.
 *
 * null ou une chaîne vide permettent de supprimer la biographie.
 */
function normalizeBio(value) {
  if (value === null) {
    return null;
  }

  if (typeof value !== "string") {
    throw new HttpError(
      400,
      "La biographie doit être une chaîne de caractères."
    );
  }

  // Suppression des espaces placés au début et à la fin.
  const normalizedValue = value.trim();

  if (normalizedValue.length > 160) {
    throw new HttpError(
      400,
      "La biographie ne doit pas dépasser 160 caractères."
    );
  }

  // Une chaîne vide est enregistrée sous la forme null.
  return normalizedValue.length === 0
    ? null
    : normalizedValue;
}

/**
 * Valide et normalise l'URL de l'avatar.
 *
 * null ou une chaîne vide permettent de supprimer l'avatar.
 */
function normalizeAvatarUrl(value) {
  if (value === null) {
    return null;
  }

  if (typeof value !== "string") {
    throw new HttpError(
      400,
      "L'URL de l'avatar doit être une chaîne de caractères."
    );
  }

  const normalizedValue = value.trim();

  // Une URL vide correspond à l'absence d'avatar.
  if (normalizedValue.length === 0) {
    return null;
  }

  if (normalizedValue.length > 500) {
    throw new HttpError(
      400,
      "L'URL de l'avatar ne doit pas dépasser 500 caractères."
    );
  }

  if (!isValidHttpUrl(normalizedValue)) {
    throw new HttpError(
      400,
      "L'URL de l'avatar doit être une URL HTTP ou HTTPS valide."
    );
  }

  return normalizedValue;
}

/**
 * Retourne le profil appartenant à l'utilisateur connecté.
 */
export async function getProfile(userId) {
  const profile = await findProfileByUserId(userId);

  if (!profile) {
    throw new HttpError(
      404,
      "Profil introuvable."
    );
  }

  return profile;
}

/**
 * Retourne les profils publics correspondant à une liste d'identifiants.
 *
 * Seules les informations destinées à l'affichage public sont renvoyées
 * (identifiant, nom affiché, avatar). Les identifiants mal formés sont
 * ignorés silencieusement.
 *
 * @param {string} idsParam Identifiants séparés par des virgules.
 * @returns {Promise<object[]>} Profils publics trouvés.
 */
export async function getPublicProfiles(idsParam) {
  if (typeof idsParam !== "string" || idsParam.trim().length === 0) {
    throw new HttpError(
      400,
      "Le paramètre ids est obligatoire."
    );
  }

  // Découpage, nettoyage, validation et déduplication des identifiants.
  const ids = [
    ...new Set(
      idsParam
        .split(",")
        .map((value) => value.trim())
        .filter((value) => UUID_PATTERN.test(value))
    )
  ];

  if (ids.length === 0) {
    return [];
  }

  if (ids.length > MAX_BATCH_SIZE) {
    throw new HttpError(
      400,
      `Le nombre d'identifiants ne doit pas dépasser ${MAX_BATCH_SIZE}.`
    );
  }

  return findPublicProfilesByUserIds(ids);
}

/**
 * Modifie partiellement le profil de l'utilisateur connecté.
 */
export async function updateProfile(
  userId,
  payload
) {
  // Vérification du format général du corps JSON.
  if (
    !payload ||
    typeof payload !== "object" ||
    Array.isArray(payload)
  ) {
    throw new HttpError(
      400,
      "Le corps de la requête est invalide."
    );
  }

  validateAllowedFields(payload);

  // Chargement du profil actuel avant d'appliquer le PATCH.
  const currentProfile = await findProfileByUserId(userId);

  if (!currentProfile) {
    throw new HttpError(
      404,
      "Profil introuvable."
    );
  }

  /*
   * Chaque valeur absente conserve la valeur actuelle.
   * Chaque valeur présente est validée et normalisée.
   */
  const nextDisplayName =
    payload.displayName === undefined
      ? currentProfile.displayName
      : normalizeDisplayName(payload.displayName);

  const nextBio =
    payload.bio === undefined
      ? currentProfile.bio
      : normalizeBio(payload.bio);

  const nextAvatarUrl =
    payload.avatarUrl === undefined
      ? currentProfile.avatarUrl
      : normalizeAvatarUrl(payload.avatarUrl);

  const updatedProfile = await updateProfileByUserId({
    userId,
    displayName: nextDisplayName,
    bio: nextBio,
    avatarUrl: nextAvatarUrl
  });

  if (!updatedProfile) {
    throw new HttpError(
      404,
      "Profil introuvable."
    );
  }

  return updatedProfile;
}