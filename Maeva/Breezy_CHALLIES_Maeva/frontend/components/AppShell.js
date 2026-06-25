"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";

import Sidebar from "./Sidebar";
import { authApi, getToken, clearToken } from "../services/api";

/*
 * Cadre commun aux pages connectées : barre latérale, contenu principal
 * et panneau droit optionnel. Il vérifie la présence du jeton et gère
 * la déconnexion. Le rendu enfant n'apparaît qu'une fois la présence
 * du jeton confirmée, pour éviter un flash de contenu protégé.
 */
export default function AppShell({ children, right = null }) {
  const router = useRouter();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (!getToken()) {
      router.replace("/login");
      return;
    }
    setReady(true);
  }, [router]);

  const handleLogout = useCallback(async () => {
    try {
      await authApi.logout();
    } catch {
      // Déconnexion sans état côté serveur : l'échec est sans conséquence.
    }
    clearToken();
    router.replace("/login");
  }, [router]);

  if (!ready) {
    return null;
  }

  return (
    <div className={`layout${right ? " with-right" : ""}`}>
      <Sidebar onLogout={handleLogout} />
      <main className="main">{children}</main>
      {right ? <aside className="right-panel">{right}</aside> : null}
    </div>
  );
}
