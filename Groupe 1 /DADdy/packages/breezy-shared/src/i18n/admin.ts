import type { Locale } from './translations'

// Chaînes propres à la modération (menu admin + signalement). Isolées ici pour
// ne pas dupliquer dans les 11 locales du fichier principal : on couvre fr et en
// et on retombe sur l'anglais pour les autres langues.

const en = {
  menu: 'Moderation',
  title: 'Moderation',
  tabReports: 'Reports',
  tabUsers: 'Users',
  // Reports
  filterAll: 'All',
  filterPending: 'Pending',
  filterResolved: 'Resolved',
  filterRejected: 'Rejected',
  noReports: 'No reports here.',
  reason: 'Reason',
  reportedPostBy: 'Post by',
  deletedPost: 'Post no longer available',
  actionResolve: 'Keep',
  actionReject: 'Reject',
  actionDelete: 'Delete post',
  viewPost: 'View post',
  // Users
  searchUsers: 'Search users…',
  noUsers: 'No users found.',
  ban: 'Ban',
  suspend: 'Suspend',
  lift: 'Reinstate',
  statusActive: 'Active',
  statusBanned: 'Banned',
  statusSuspended: 'Suspended',
  sanctionTitleBan: 'Ban user',
  sanctionTitleSuspend: 'Suspend user',
  sanctionReasonPlaceholder: 'Reason (optional)…',
  confirm: 'Confirm',
  cancel: 'Cancel',
  working: 'Working…',
  // Report (user side)
  report: 'Report',
  reportTitle: 'Report this post',
  reportReasonPlaceholder: 'Why are you reporting this post?',
  reportSubmit: 'Report',
  reportSubmitting: 'Sending…',
  reported: 'Reported',
  reportError: 'Could not send the report.',
  alreadyReported: 'You already reported this post.',
}

const fr: typeof en = {
  menu: 'Modération',
  title: 'Modération',
  tabReports: 'Signalements',
  tabUsers: 'Utilisateurs',
  filterAll: 'Tous',
  filterPending: 'En attente',
  filterResolved: 'Traités',
  filterRejected: 'Rejetés',
  noReports: 'Aucun signalement ici.',
  reason: 'Motif',
  reportedPostBy: 'Post de',
  deletedPost: 'Post non disponible',
  actionResolve: 'Conserver',
  actionReject: 'Rejeter',
  actionDelete: 'Supprimer le post',
  viewPost: 'Voir le post',
  searchUsers: 'Rechercher des utilisateurs…',
  noUsers: 'Aucun utilisateur trouvé.',
  ban: 'Bannir',
  suspend: 'Suspendre',
  lift: 'Réintégrer',
  statusActive: 'Actif',
  statusBanned: 'Banni',
  statusSuspended: 'Suspendu',
  sanctionTitleBan: "Bannir l'utilisateur",
  sanctionTitleSuspend: "Suspendre l'utilisateur",
  sanctionReasonPlaceholder: 'Motif (optionnel)…',
  confirm: 'Confirmer',
  cancel: 'Annuler',
  working: 'En cours…',
  report: 'Signaler',
  reportTitle: 'Signaler ce post',
  reportReasonPlaceholder: 'Pourquoi signalez-vous ce post ?',
  reportSubmit: 'Signaler',
  reportSubmitting: 'Envoi…',
  reported: 'Signalé',
  reportError: "Impossible d'envoyer le signalement.",
  alreadyReported: 'Vous avez déjà signalé ce post.',
}

export type AdminStrings = typeof en

export function adminT(locale: Locale): AdminStrings {
  return locale === 'fr' ? fr : en
}
