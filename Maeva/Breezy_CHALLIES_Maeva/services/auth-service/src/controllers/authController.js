import {
  getAuthenticatedUser,
  loginUser,
  registerUser
} from "../services/authService.js";

/**
 * Crée un compte à partir du corps de la requête puis renvoie les
 * informations publiques de l'utilisateur (sans le mot de passe).
 */
export async function register(req, res, next) {
  try {
    const user = await registerUser(req.body);

    return res.status(201).json({
      message: "Compte créé.",

      user: {
        id: user.id,
        email: user.email,
        role: user.role
      }
    });
  } catch (error) {
    return next(error);
  }
}

/**
 * Authentifie l'utilisateur et renvoie un jeton d'accès (JWT) ainsi que
 * ses informations publiques.
 */
export async function login(req, res, next) {
  try {
    const result = await loginUser(req.body);

    return res.status(200).json(result);
  } catch (error) {
    return next(error);
  }
}

/**
 * Renvoie l'utilisateur courant, identifié par le `sub` du JWT vérifié.
 */
export async function me(req, res, next) {
  try {
    // L'identifiant provient du jeton, jamais de la requête elle-même.
    const user = await getAuthenticatedUser(
      req.user.sub
    );

    return res.status(200).json(user);
  } catch (error) {
    return next(error);
  }
}

/*
 * Déconnexion sans état : le JWT n'étant pas stocké côté serveur, il n'y a
 * rien à invalider. On renvoie simplement un accusé de réception ; le
 * frontend se charge d'oublier le jeton.
 */
export function logout(_req, res) {
  return res.status(200).json({
    message: "Déconnexion effectuée."
  });
}