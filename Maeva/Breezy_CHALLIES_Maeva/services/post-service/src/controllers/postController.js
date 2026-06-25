import {
  createPost,
  listFeed,
  listUserPosts
} from "../services/postService.js";

/**
 * Crée une publication pour l'utilisateur authentifié.
 */
export async function publishPost(req, res, next) {
  try {
    // L’auteur provient toujours du JWT, jamais du corps de la requête.
    const post = await createPost({
      authorId: req.user.sub,
      content: req.body?.content
    });

    return res.status(201).json(post);
  } catch (error) {
    return next(error);
  }
}

/**
 * Renvoie les publications de l'utilisateur authentifié (page profil).
 */
export async function listMyPosts(req, res, next) {
  try {
    const posts = await listUserPosts(req.user.sub);

    return res.status(200).json(posts);
  } catch (error) {
    return next(error);
  }
}

/**
 * Renvoie le fil d'actualité global (toutes les publications).
 */
export async function listAllPosts(_req, res, next) {
  try {
    const posts = await listFeed();

    return res.status(200).json(posts);
  } catch (error) {
    return next(error);
  }
}
