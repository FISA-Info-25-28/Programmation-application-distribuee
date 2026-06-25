"use client";

import { useState } from "react";

import Avatar from "./Avatar";
import { postsApi, ApiError } from "../services/api";

const MAX_POST_LENGTH = 280;

/*
 * Zone de publication : « Comment ça va ? ». Compteur de caractères,
 * bouton désactivé tant que le contenu est vide, trop long ou en cours
 * d'envoi. Notifie le parent du nouveau post via onPublished.
 */
export default function Composer({ author, onPublished, onApiError }) {
  const [content, setContent] = useState("");
  const [error, setError] = useState(null);
  const [publishing, setPublishing] = useState(false);

  const over = content.length > MAX_POST_LENGTH;
  const canPublish = !publishing && content.trim().length > 0 && !over;

  async function handleSubmit(event) {
    event.preventDefault();
    if (!canPublish) {
      return;
    }

    setError(null);
    setPublishing(true);

    try {
      const post = await postsApi.create(content.trim());
      onPublished(post);
      setContent("");
    } catch (err) {
      if (onApiError && onApiError(err)) {
        return;
      }
      setError(
        err instanceof ApiError ? err.message : "Publication impossible."
      );
    } finally {
      setPublishing(false);
    }
  }

  return (
    <section className="card">
      {error ? <div className="error">{error}</div> : null}

      <form className="composer" onSubmit={handleSubmit}>
        <Avatar
          avatarUrl={author?.avatarUrl}
          displayName={author?.displayName}
          size="sm"
        />

        <div className="composer-body">
          <textarea
            aria-label="Comment ça va ?"
            placeholder="Comment ça va ?"
            value={content}
            onChange={(e) => setContent(e.target.value)}
          />

          <div className="composer-footer">
            <span className={`counter${over ? " over" : ""}`}>
              {content.length} / {MAX_POST_LENGTH}
            </span>
            <button
              type="submit"
              className="btn-dark"
              disabled={!canPublish}
            >
              {publishing ? "Publication…" : "Publier"}
            </button>
          </div>
        </div>
      </form>
    </section>
  );
}
