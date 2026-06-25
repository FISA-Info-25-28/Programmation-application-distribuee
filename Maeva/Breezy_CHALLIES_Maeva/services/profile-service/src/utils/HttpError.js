/**
 * Erreur métier contenant directement le code HTTP
 * devant être retourné au client.
 */
export class HttpError extends Error {
  /**
   * @param {number} statusCode Code HTTP de l'erreur.
   * @param {string} message Message destiné au client.
   */
  constructor(statusCode, message) {
    super(message);

    // Nom explicite utile dans les logs.
    this.name = "HttpError";

    // Code HTTP utilisé par le middleware d'erreur.
    this.statusCode = statusCode;
  }
}