import {
  getProfile,
  getPublicProfiles,
  updateProfile
} from "../services/profileService.js";

/**
 * Retourne les profils publics (nom, avatar) d'une liste d'identifiants.
 *
 * Réservé aux utilisateurs authentifiés ; sert à enrichir le fil
 * d'actualité côté frontend.
 */
export async function getProfilesBatch(req, res, next) {
  try {
    const profiles = await getPublicProfiles(req.query.ids);

    return res.status(200).json(profiles);
  } catch (error) {
    return next(error);
  }
}

/**
 * Retourne le profil de l'utilisateur authentifié.
 */
export async function getMyProfile(
  req,
  res,
  next
) {
  try {
    // L'identifiant est issu du JWT vérifié.
    const userId = req.user.sub;

    const profile = await getProfile(userId);

    return res.status(200).json(profile);
  } catch (error) {
    // Transmission au middleware centralisé de gestion des erreurs.
    return next(error);
  }
}

/**
 * Modifie le profil de l'utilisateur authentifié.
 */
export async function updateMyProfile(
  req,
  res,
  next
) {
  try {
    // L'utilisateur ne peut pas choisir l'identifiant du profil.
    const userId = req.user.sub;

    const profile = await updateProfile(
      userId,
      req.body
    );

    return res.status(200).json({
      message: "Profil mis à jour.",
      profile
    });
  } catch (error) {
    return next(error);
  }
}