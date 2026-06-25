import { isAxiosError } from 'axios'
import type { ApiError } from '../types/api'

// Catégories d'erreurs renvoyées par l'auth-service sur les flux liés au mot de
// passe (inscription, réinitialisation, changement). Le back ne renvoie qu'un
// seul code `VALIDATION_ERROR` : on classe donc à partir du message (texte
// canonique, défini par les constantes côté auth-service/validation.go).
export type PasswordErrorKind =
  | 'tooCommon'
  | 'length'
  | 'composition'
  | 'identifier'
  | 'sameAsCurrent'
  | 'currentIncorrect'
  | 'invalidToken'
  | 'usernameLength'
  | 'invalidEmail'
  | 'conflict'
  | 'network'
  | 'generic'

// Fragments stables des messages serveur. La comparaison se fait en minuscules et
// par sous-chaîne, donc robuste à un éventuel préfixe ("validation: ...").
const MESSAGE_PATTERNS: ReadonlyArray<[string, PasswordErrorKind]> = [
  ['too common', 'tooCommon'],
  ['between 12 and 72', 'length'],
  ['at least 3 of', 'composition'],
  ['must not contain your username', 'identifier'],
  ['differ from the current', 'sameAsCurrent'],
  ['current password is incorrect', 'currentIncorrect'],
  ['reset token', 'invalidToken'],
  ['username must be between', 'usernameLength'],
  ['invalid email', 'invalidEmail'],
]

export function classifyPasswordError(error: unknown): PasswordErrorKind {
  if (!isAxiosError<ApiError>(error) || !error.response) return 'network'

  const { status, data } = error.response
  if (status === 409) return 'conflict'
  if (status >= 500) return 'network'

  const message = (data?.message ?? '').toLowerCase()
  for (const [fragment, kind] of MESSAGE_PATTERNS) {
    if (message.includes(fragment)) return kind
  }
  return 'generic'
}

// Messages flash en français (l'app est francophone par défaut ; ResetPasswordPage
// et ChangePasswordForm utilisent déjà des libellés FR en dur).
const MESSAGES: Record<PasswordErrorKind, string> = {
  tooCommon: 'Ce mot de passe est trop courant. Choisissez-en un moins prévisible.',
  length: 'Le mot de passe doit contenir entre 12 et 72 caractères.',
  composition:
    'Le mot de passe doit combiner au moins 3 catégories parmi : minuscule, majuscule, chiffre et caractère spécial.',
  identifier: "Le mot de passe ne doit pas contenir votre nom d'utilisateur ni votre email.",
  sameAsCurrent: "Le nouveau mot de passe doit être différent de l'actuel.",
  currentIncorrect: 'Votre mot de passe actuel est incorrect.',
  invalidToken: 'Ce lien de réinitialisation est invalide ou expiré. Demandez-en un nouveau.',
  usernameLength: "Le nom d'utilisateur doit contenir entre 3 et 50 caractères.",
  invalidEmail: "L'adresse email est invalide.",
  conflict: 'Un compte existe déjà avec ce nom ou cet email.',
  network: 'Action impossible pour le moment. Réessayez plus tard.',
  generic: 'Vérifiez vos informations et réessayez.',
}

// passwordErrorMessage renvoie le message flash prêt à afficher pour une erreur
// remontée par l'un des flux mot de passe.
export function passwordErrorMessage(error: unknown): string {
  return MESSAGES[classifyPasswordError(error)]
}

// Indice affiché sous les champs mot de passe : résume les règles appliquées
// côté serveur (cf. auth-service/validation.go) pour guider l'utilisateur avant
// soumission.
export const passwordRequirementsHint =
  '12 caractères minimum, avec au moins 3 types parmi : minuscule, majuscule, chiffre et caractère spécial.'
