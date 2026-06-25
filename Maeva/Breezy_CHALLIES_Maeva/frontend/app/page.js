"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";

import Logo from "../components/Logo";
import { getToken } from "../services/api";

/*
 * Page d'accueil publique (« / ») : présente la marque et propose de se
 * connecter ou de s'inscrire. Un visiteur déjà authentifié est renvoyé
 * directement vers son fil d'actualité.
 */
export default function LandingPage() {
  const router = useRouter();

  // Un visiteur déjà connecté va directement sur son fil d'accueil.
  useEffect(() => {
    if (getToken()) {
      router.replace("/home");
    }
  }, [router]);

  return (
    <main className="split">
      <div className="split-brand">
        <Logo large />
      </div>

      <div className="split-actions">
        <Link href="/login">
          <button className="btn-dark">S’authentifier</button>
        </Link>
        <Link href="/register">
          <button className="btn-outline">Créer un compte</button>
        </Link>
      </div>
    </main>
  );
}
