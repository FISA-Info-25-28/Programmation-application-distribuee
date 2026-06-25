/*
 * Client HTTP unique du frontend.
 *
 * Le frontend ne connaît que l'API Gateway : il ignore les ports
 * et les noms internes des trois services. Toutes les requêtes
 * passent par NEXT_PUBLIC_API_URL (http://localhost:8080/api).
 */

const API_URL =
  process.env.NEXT_PUBLIC_API_URL ||
  "http://localhost:8080/api";

// Le JWT est stocké côté client (Bearer token, simplification assumée du MVP).
const TOKEN_KEY = "breezy_token";

/**
 * Erreur enrichie du code HTTP, afin que les pages puissent
 * distinguer un 401 (redirection) d'une simple erreur de validation.
 */
export class ApiError extends Error {
  constructor(status, message) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

/* ------------------------------------------------------------------ */
/* Gestion du jeton                                                    */
/* ------------------------------------------------------------------ */

export function getToken() {
  if (typeof window === "undefined") {
    return null;
  }

  return window.localStorage.getItem(TOKEN_KEY);
}

export function setToken(token) {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  if (typeof window === "undefined") {
    return;
  }

  window.localStorage.removeItem(TOKEN_KEY);
}

/* ------------------------------------------------------------------ */
/* Requête générique                                                   */
/* ------------------------------------------------------------------ */

async function request(
  path,
  { method = "GET", body, auth = false } = {}
) {
  const headers = {};

  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }

  if (auth) {
    const token = getToken();

    if (token) {
      headers.Authorization = `Bearer ${token}`;
    }
  }

  let response;

  try {
    response = await fetch(`${API_URL}${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined
    });
  } catch (networkError) {
    // Le serveur est injoignable (gateway arrêté, etc.).
    throw new ApiError(
      0,
      "Impossible de contacter le serveur. Réessayez plus tard."
    );
  }

  // Certaines réponses (204) ou erreurs n'ont pas de corps JSON.
  let data = null;

  const text = await response.text();

  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = null;
    }
  }

  if (!response.ok) {
    const message =
      (data && data.message) ||
      "Une erreur est survenue.";

    throw new ApiError(response.status, message);
  }

  return data;
}

/* ------------------------------------------------------------------ */
/* Authentification                                                    */
/* ------------------------------------------------------------------ */

export const authApi = {
  register({ email, password, displayName }) {
    return request("/auth/register", {
      method: "POST",
      body: { email, password, displayName }
    });
  },

  login({ email, password }) {
    return request("/auth/login", {
      method: "POST",
      body: { email, password }
    });
  },

  logout() {
    return request("/auth/logout", {
      method: "POST",
      auth: true
    });
  },

  me() {
    return request("/auth/me", { auth: true });
  }
};

/* ------------------------------------------------------------------ */
/* Profil                                                              */
/* ------------------------------------------------------------------ */

export const profileApi = {
  me() {
    return request("/profile/me", { auth: true });
  },

  // Résout les profils publics (nom, avatar) d'une liste d'identifiants.
  batch(ids) {
    const query = encodeURIComponent(ids.join(","));
    return request(`/profile/batch?ids=${query}`, { auth: true });
  },

  update(payload) {
    return request("/profile/me", {
      method: "PATCH",
      body: payload,
      auth: true
    });
  }
};

/* ------------------------------------------------------------------ */
/* Publications                                                        */
/* ------------------------------------------------------------------ */

export const postsApi = {
  // Publications de l'utilisateur connecté (page profil).
  list() {
    return request("/posts/me", { auth: true });
  },

  // Fil d'actualité global. Slash final : voir la note sur create().
  feed() {
    return request("/posts/", { auth: true });
  },

  create(content) {
    // Slash final : le gateway ne route que /api/posts/ et renverrait
    // une redirection 301 (perdant le port) sur la forme sans slash.
    return request("/posts/", {
      method: "POST",
      body: { content },
      auth: true
    });
  }
};
