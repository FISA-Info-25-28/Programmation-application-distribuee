/**
 * Protège les routes internes (service à service) par une clé partagée.
 *
 * Seul l'Auth Service, qui connaît INTERNAL_API_KEY, peut appeler ces
 * routes ; elles ne sont pas exposées via la gateway publique.
 */
export function verifyInternalApiKey(
  req,
  res,
  next
) {
  // Clé transmise par l'appelant dans l'en-tête dédié.
  const receivedApiKey =
    req.headers["x-internal-api-key"];

  // Clé attendue, fournie par l'environnement.
  const expectedApiKey =
    process.env.INTERNAL_API_KEY;

  // Refus si la clé est absente côté serveur ou ne correspond pas.
  if (
    !expectedApiKey ||
    receivedApiKey !== expectedApiKey
  ) {
    return res.status(401).json({
      message: "Clé interne invalide."
    });
  }

  return next();
}