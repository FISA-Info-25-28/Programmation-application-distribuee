import "./globals.css";

// Métadonnées du document, partagées par toutes les pages (onglet, SEO).
export const metadata = {
  title: "Breezy",
  description: "Réseau social léger et distribué"
};

/*
 * Layout racine de l'application Next.js : il enveloppe toutes les pages
 * et définit la structure HTML commune (langue, body, styles globaux).
 */
export default function RootLayout({ children }) {
  return (
    <html lang="fr">
      <body>{children}</body>
    </html>
  );
}
