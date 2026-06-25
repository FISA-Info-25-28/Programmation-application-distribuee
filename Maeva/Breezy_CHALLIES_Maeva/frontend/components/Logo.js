/*
 * Logo Breezy : une icône « vent » (trois traits dont deux s'enroulent)
 * accompagnée du nom. La variante `large` est utilisée sur les écrans
 * d'authentification.
 */
export default function Logo({ large = false }) {
  return (
    <span className={`logo${large ? " logo-lg" : ""}`}>
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d="M3 8h11a3 3 0 1 0-3-3" />
        <path d="M3 12h15a3 3 0 1 1-3 3" />
        <path d="M3 16h9" />
      </svg>
      <span>Breezy</span>
    </span>
  );
}
