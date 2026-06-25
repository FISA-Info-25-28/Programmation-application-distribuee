/*
 * Avatar : image fournie par avatarUrl, ou à défaut l'initiale du nom
 * sur une pastille. `size` vaut "sm" (posts) ou "lg" (en-tête de profil).
 */
export default function Avatar({ avatarUrl, displayName, size = "sm" }) {
  const className = `avatar avatar-${size}`;

  if (avatarUrl) {
    return (
      <img
        className={className}
        src={avatarUrl}
        alt={`Avatar de ${displayName || "l'utilisateur"}`}
      />
    );
  }

  const initial = (displayName || "?").trim().charAt(0).toUpperCase();

  return <div className={className}>{initial}</div>;
}
