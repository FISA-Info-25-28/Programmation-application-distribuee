import jwt from "jsonwebtoken";

/**
 * Vérifie le JWT transmis dans l'en-tête Authorization.
 *
 * Format attendu :
 * Authorization: Bearer <token>
 */
export function authenticate(req, res, next) {
  // Récupération de l'en-tête HTTP d'authentification.
  const authorizationHeader = req.headers.authorization;

  // Refus lorsque l'en-tête est absent ou mal formé.
  if (
    !authorizationHeader ||
    !authorizationHeader.startsWith("Bearer ")
  ) {
    return res.status(401).json({
      message: "Authentification requise."
    });
  }

  // Suppression du préfixe "Bearer " pour conserver le JWT.
  const token = authorizationHeader.slice("Bearer ".length);

  try {
    // Vérification de la signature et de la date d'expiration.
    const payload = jwt.verify(
      token,
      process.env.JWT_SECRET
    );

    // Un JWT valide pour Breezy doit contenir un champ sub.
    if (
      typeof payload === "string" ||
      !payload.sub
    ) {
      return res.status(401).json({
        message: "Jeton invalide."
      });
    }

    // Les informations utiles sont ajoutées à la requête.
    // Le contrôleur utilisera req.user.sub comme identifiant.
    req.user = {
      sub: payload.sub,
      email: payload.email,
      role: payload.role
    };

    // Passage au contrôleur suivant.
    return next();
  } catch {
    // Cette réponse couvre notamment les JWT expirés ou falsifiés.
    return res.status(401).json({
      message: "Jeton invalide ou expiré."
    });
  }
}