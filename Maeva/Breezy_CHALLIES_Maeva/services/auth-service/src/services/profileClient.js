/*
 * Petit client HTTP vers la route interne du Profile Service. L'Auth
 * Service l'appelle au moment de l'inscription pour créer le profil
 * associé au nouveau compte (communication service à service).
 */

// URL interne du Profile Service (réseau Docker), hors gateway public.
const profileServiceUrl =
  process.env.PROFILE_SERVICE_INTERNAL_URL;

// Clé partagée authentifiant l'appel entre services internes.
const internalApiKey =
  process.env.INTERNAL_API_KEY;

/*
 * Crée le profil du nouvel utilisateur. Lève une erreur si la
 * configuration est manquante ou si le Profile Service répond en échec,
 * ce qui déclenche la compensation côté authService (suppression du compte).
 */
export async function createProfile({
  userId,
  displayName
}) {
  if (!profileServiceUrl) {
    throw new Error(
      "PROFILE_SERVICE_INTERNAL_URL est manquante."
    );
  }

  if (!internalApiKey) {
    throw new Error(
      "INTERNAL_API_KEY est manquante."
    );
  }

  const response = await fetch(
    `${profileServiceUrl}/internal/profiles`,
    {
      method: "POST",

      headers: {
        "Content-Type": "application/json",
        "X-Internal-Api-Key": internalApiKey
      },

      body: JSON.stringify({
        userId,
        displayName
      }),

      // Évite de bloquer l'inscription si le Profile Service ne répond pas.
      signal: AbortSignal.timeout(5000)
    }
  );

  if (!response.ok) {
    const responseBody = await response.text();

    throw new Error(
      `Profile Service inaccessible : HTTP ${response.status} ${responseBody}`
    );
  }

  return response.json();
}