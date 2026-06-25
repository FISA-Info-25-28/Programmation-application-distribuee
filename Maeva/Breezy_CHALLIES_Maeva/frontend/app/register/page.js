"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";

import Logo from "../../components/Logo";
import { authApi, ApiError } from "../../services/api";

/*
 * Page d'inscription : contrôle les champs côté client (champs requis,
 * longueur du mot de passe, confirmation) puis crée le compte. L'API ne
 * renvoyant pas de jeton, on redirige ensuite vers la page de connexion.
 */
export default function RegisterPage() {
  const router = useRouter();

  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");

  const [error, setError] = useState(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event) {
    event.preventDefault();
    setError(null);

    if (!displayName.trim() || !email.trim() || !password) {
      setError("Tous les champs sont obligatoires.");
      return;
    }

    if (password.length < 8) {
      setError("Le mot de passe doit contenir au moins 8 caractères.");
      return;
    }

    if (password !== confirm) {
      setError("Les mots de passe ne correspondent pas.");
      return;
    }

    setSubmitting(true);

    try {
      await authApi.register({
        displayName: displayName.trim(),
        email: email.trim(),
        password
      });

      // L'inscription ne renvoie pas de jeton : on redirige vers la connexion.
      router.push("/login?registered=1");
    } catch (err) {
      const message =
        err instanceof ApiError
          ? err.message
          : "Inscription impossible. Réessayez.";
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
        <h1>Créer un compte</h1>

        {error ? <div className="error">{error}</div> : null}

        <form onSubmit={handleSubmit} noValidate>
          <div className="field">
            <label htmlFor="displayName">Nom d’affichage</label>
            <input
              id="displayName"
              type="text"
              maxLength={50}
              placeholder="écrire ici"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              autoComplete="name"
            />
          </div>

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
              placeholder="8 caractères minimum"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="new-password"
            />
          </div>

          <div className="field">
            <label htmlFor="confirm">Confirmation du mot de passe</label>
            <input
              id="confirm"
              type="password"
              placeholder="écrire ici"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              autoComplete="new-password"
            />
          </div>

          <button
            type="submit"
            className="btn-dark btn-block"
            disabled={submitting}
          >
            {submitting ? "Création…" : "Créer mon compte"}
          </button>
        </form>

        <p className="muted" style={{ marginTop: 18, marginBottom: 0 }}>
          Déjà inscrit ?{" "}
          <Link href="/login" style={{ textDecoration: "underline" }}>
            Se connecter
          </Link>
        </p>
      </section>
    </main>
  );
}
