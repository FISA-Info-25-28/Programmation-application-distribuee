"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";

import Logo from "../../components/Logo";
import { authApi, setToken, ApiError } from "../../services/api";

/*
 * Page de connexion : valide la saisie, appelle l'API d'authentification
 * puis stocke le jeton avant de rediriger vers le fil d'accueil. Affiche
 * une bannière de succès lorsqu'on arrive depuis l'inscription.
 */
export default function LoginPage() {
  const router = useRouter();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  const [error, setError] = useState(null);
  const [registered, setRegistered] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  // Bannière de succès après une inscription (?registered=1).
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get("registered") === "1") {
      setRegistered(true);
    }
  }, []);

  async function handleSubmit(event) {
    event.preventDefault();
    setError(null);

    if (!email.trim() || !password) {
      setError("Courriel et mot de passe sont obligatoires.");
      return;
    }

    setSubmitting(true);

    try {
      const result = await authApi.login({
        email: email.trim(),
        password
      });

      setToken(result.accessToken);
      router.push("/home");
    } catch (err) {
      const message =
        err instanceof ApiError
          ? err.message
          : "Connexion impossible. Réessayez.";
      setError(message);
      setSubmitting(false);
    }
  }

  return (
    <main className="split">
      <div className="split-brand">
        <Logo large />
      </div>

      <section className="split-panel">
        <h1>Connexion</h1>

        {registered ? (
          <div className="success">
            Compte créé. Vous pouvez maintenant vous connecter.
          </div>
        ) : null}

        {error ? <div className="error">{error}</div> : null}

        <form onSubmit={handleSubmit} noValidate>
          <div className="field">
            <label htmlFor="email">Courriel</label>
            <input
              id="email"
              type="email"
              placeholder="écrire ici"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
            />
          </div>

          <div className="field">
            <label htmlFor="password">Mot de passe</label>
            <input
              id="password"
              type="password"
              placeholder="écrire ici"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
            />
          </div>

          <button
            type="submit"
            className="btn-dark btn-block"
            disabled={submitting}
          >
            {submitting ? "Connexion…" : "Se connecter"}
          </button>
        </form>

        <p className="muted" style={{ marginTop: 18, marginBottom: 0 }}>
          Pas encore de compte ?{" "}
          <Link href="/register" style={{ textDecoration: "underline" }}>
            Créer un compte
          </Link>
        </p>
      </section>
    </main>
  );
}
