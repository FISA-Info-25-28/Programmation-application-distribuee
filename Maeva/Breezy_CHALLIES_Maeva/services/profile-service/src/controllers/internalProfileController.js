import { createProfile } from "../repositories/profileRepository.js";

// Format d'un UUID v1-v5, pour valider l'identifiant reçu de l'Auth Service.
const UUID_PATTERN =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

/**
 * Crée un profil à la demande de l'Auth Service (route interne, protégée
 * par clé API). Valide l'identifiant et le nom d'affichage, puis traite le
 * cas du doublon (profil déjà créé) renvoyé par PostgreSQL.
 */
export async function createInternalProfile(
  req,
  res
) {
  const {
    userId,
    displayName
  } = req.body;

  if (
    typeof userId !== "string" ||
    !UUID_PATTERN.test(userId)
  ) {
    return res.status(400).json({
      message: "L'identifiant utilisateur est invalide."
    });
  }

  if (
    typeof displayName !== "string" ||
    displayName.trim().length === 0 ||
    displayName.trim().length > 50
  ) {
    return res.status(400).json({
      message: "Le nom d'affichage est invalide."
    });
  }

  try {
    const profile = await createProfile({
      userId,
      displayName: displayName.trim()
    });

    return res.status(201).json(profile);
  } catch (error) {
    // 23505 = violation d'unicité : le profil existe déjà pour cet userId.
    if (error.code === "23505") {
      return res.status(409).json({
        message: "Ce profil existe déjà."
      });
    }

    console.error(
      "Erreur de création du profil :",
      error
    );

    return res.status(500).json({
      message: "La création du profil a échoué."
    });
  }
}