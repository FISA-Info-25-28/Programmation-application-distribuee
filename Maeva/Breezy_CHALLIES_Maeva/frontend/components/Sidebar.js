"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import Logo from "./Logo";
import {
  HomeIcon,
  ProfileIcon,
  BellIcon,
  MailIcon,
  LogoutIcon
} from "./icons";

/*
 * Barre latérale de navigation des pages connectées. Met en évidence
 * l'onglet courant (via le chemin actif) et expose le bouton de
 * déconnexion. Les onglets Notifications et Messages sont hors périmètre
 * du MVP : ils restent affichés mais désactivés pour rester fidèle aux
 * maquettes.
 */
export default function Sidebar({ onLogout }) {
  // Chemin courant, utilisé pour marquer le lien actif.
  const pathname = usePathname();

  return (
    <nav className="sidebar">
      <Link href="/home" aria-label="Accueil Breezy">
        <Logo />
      </Link>

      <Link
        href="/home"
        className={`nav-item${pathname === "/home" ? " active" : ""}`}
      >
        <HomeIcon />
        <span className="label">Fil d'actualité</span>
      </Link>

      <Link
        href="/profile"
        className={`nav-item${pathname === "/profile" ? " active" : ""}`}
      >
        <ProfileIcon />
        <span className="label">Profil</span>
      </Link>

      {/* Hors périmètre du MVP : présents pour la fidélité visuelle. */}
      <span
        className="nav-item disabled"
        title="Hors périmètre du MVP"
        aria-disabled="true"
      >
        <BellIcon />
        <span className="label">Notifications</span>
      </span>

      <span
        className="nav-item disabled"
        title="Hors périmètre du MVP"
        aria-disabled="true"
      >
        <MailIcon />
        <span className="label">Messages</span>
      </span>

      <div className="sidebar-spacer" />

      <button className="nav-item" onClick={onLogout} type="button">
        <LogoutIcon />
        <span className="label">Se déconnecter</span>
      </button>
    </nav>
  );
}
