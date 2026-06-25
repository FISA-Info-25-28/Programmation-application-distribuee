import { TERMS_VERSION } from './terms'

export function createRegisterRequest(
  username: string,
  email: string,
  password: string,
  termsAccepted: boolean,
) {
  return {
    username,
    email,
    password,
    termsAccepted,
    termsVersion: TERMS_VERSION,
  }
}
