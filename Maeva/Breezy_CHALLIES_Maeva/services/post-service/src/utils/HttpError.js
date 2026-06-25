// Cette classe permet de distinguer une erreur métier d’une erreur serveur.

export class HttpError extends Error {
  constructor(statusCode, message) {
    super(message);

    this.name = "HttpError";
    this.statusCode = statusCode;
  }
}
