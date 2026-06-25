/*
 * Petites fonctions de présentation partagées par les composants.
 */

const dateFormatter = new Intl.DateTimeFormat("fr-FR", {
  dateStyle: "long",
  timeStyle: "short"
});

export function formatDate(value) {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return dateFormatter.format(date);
}

/*
 * Le MVP ne gère pas de pseudo dédié : on dérive un identifiant
 * lisible (@quelquechose) à partir du nom d'affichage.
 */
export function handleFromName(displayName) {
  if (!displayName) {
    return "@utilisateur";
  }

  const slug = displayName
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]/g, "");

  return `@${slug || "utilisateur"}`;
}
