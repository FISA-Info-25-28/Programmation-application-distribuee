import { randomUUID } from "node:crypto";

import {
  findAllPosts,
  findPostsByAuthorId,
  insertPost
} from "../repositories/postRepository.js";

import { HttpError } from "../utils/HttpError.js";

export const MAX_CONTENT_LENGTH = 280;

/*
 * Valide et normalise le contenu d’un post.
 * Les espaces de début et de fin sont retirés,
 * puis la limite de 280 caractères est appliquée
 * sur le contenu nettoyé.
 */
export function normalizePostContent(content) {
  if (typeof content !== "string") {
    throw new HttpError(
      400,
      "Le contenu du post est obligatoire."
    );
  }

  const trimmedContent = content.trim();

  if (trimmedContent.length === 0) {
    throw new HttpError(
      400,
      "Le contenu du post ne peut pas être vide."
    );
  }

  if (trimmedContent.length > MAX_CONTENT_LENGTH) {
    throw new HttpError(
      400,
      `Le contenu du post ne doit pas dépasser ${MAX_CONTENT_LENGTH} caractères.`
    );
  }

  return trimmedContent;
}

export async function createPost({
  authorId,
  content
}) {
  const normalizedContent =
    normalizePostContent(content);

  const post = await insertPost({
    id: randomUUID(),
    authorId,
    content: normalizedContent
  });

  return post;
}

export async function listUserPosts(authorId) {
  return findPostsByAuthorId(authorId);
}

/*
 * Fil d’actualité global : toutes les publications, de la plus récente
 * à la plus ancienne. Le post-service ne connaît que author_id ; la
 * résolution des noms d’auteur est faite par le frontend auprès du
 * Profile Service.
 */
export async function listFeed() {
  return findAllPosts();
}
