import Avatar from "./Avatar";
import { CommentIcon, RepostIcon, HeartIcon } from "./icons";
import { formatDate, handleFromName } from "../lib/format";

/*
 * Carte d'une publication, fidèle aux maquettes : avatar, nom, identifiant,
 * date, contenu et une rangée d'actions.
 *
 * Les actions (répondre, repartager, aimer) sont volontairement décoratives :
 * elles ne sont pas couvertes par le périmètre du MVP et sont donc
 * non interactives.
 */
export default function PostCard({ post, author }) {
  const displayName = author?.displayName || "Utilisateur";

  return (
    <article className="card post">
      <Avatar
        avatarUrl={author?.avatarUrl}
        displayName={displayName}
        size="sm"
      />

      <div className="post-main">
        <div className="post-header">
          <span className="post-author">{displayName}</span>
          <span className="post-meta">{handleFromName(displayName)}</span>
          <span className="post-meta">· {formatDate(post.createdAt)}</span>
        </div>

        <p className="post-content">{post.content}</p>

        <div
          className="post-actions"
          aria-hidden="true"
          title="Fonctionnalités hors périmètre du MVP"
        >
          <span className="action">
            <CommentIcon />
          </span>
          <span className="action">
            <RepostIcon />
          </span>
          <span className="action">
            <HeartIcon />
          </span>
        </div>
      </div>
    </article>
  );
}
