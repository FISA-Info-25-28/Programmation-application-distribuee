import type { UserRole } from '../types/api'

// isStaff indique si le rôle dispose des droits de modération (modérateur ou
// administrateur). Le back renvoie 'administrator' ; 'admin' est toléré par compat.
export function isStaff(role?: UserRole | string | null): boolean {
  return role === 'moderator' || role === 'administrator' || role === 'admin'
}
