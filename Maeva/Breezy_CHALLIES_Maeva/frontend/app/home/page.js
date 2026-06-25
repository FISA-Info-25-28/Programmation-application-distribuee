"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import AppShell from "../../components/AppShell";
import Composer from "../../components/Composer";
import PostCard from "../../components/PostCard";
import {
  profileApi,
  postsApi,
  getToken,
  clearToken,
  ApiError
} from "../../services/api";

/*
 * Panneau droit « Comptes suivis » des maquettes. La gestion des
 * abonnements n'est pas couverte par le MVP : il est présenté en l'état,
 * avec une mention explicite plutôt que de fausses données.
 */
function FollowedPanel() {
  return (
    <div className="panel-card">
      <h3 style={{ margin: "0 0 8px", fontSize: "1rem" }}>Comptes suivis</h3>
      <p className="muted" style={{ margin: 0, fontSize: "0.9rem" }}>
        La gestion des abonnements n’est pas incluse dans ce MVP.
      </p>
    </div>
  );
}

export default function HomePage() {
  const router = useRouter();

  const [profile, setProfile] = useState(null);
  // Dictionnaire authorId -> { displayName, avatarUrl }.
  const [authors, setAuthors] = useState({});
  const [posts, setPosts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(null);

  const handleApiError = useCallback(
    (err) => {
      if (err instanceof ApiError && err.status === 401) {
        clearToken();
        router.replace("/login");
        return true;
      }
      return false;
    },
    [router]
  );

  // Résout le nom et l'avatar des auteurs d'une liste de posts.
  const resolveAuthors = useCallback(async (feedPosts, ownProfile) => {
    const ids = [...new Set(feedPosts.map((p) => p.authorId))];

    const map = {};

    if (ids.length > 0) {
      const profiles = await profileApi.batch(ids);
      profiles.forEach((p) => {
        map[p.userId] = p;
      });
    }

    // L'utilisateur courant est toujours présent (ses propres posts).
    if (ownProfile) {
      map[ownProfile.userId] = {
        userId: ownProfile.userId,
        displayName: ownProfile.displayName,
        avatarUrl: ownProfile.avatarUrl
      };
    }

    return map;
  }, []);

  useEffect(() => {
    if (!getToken()) {
      return;
    }

    let active = true;

    (async () => {
      try {
        const [profileData, feedData] = await Promise.all([
          profileApi.me(),
          postsApi.feed()
        ]);

        const map = await resolveAuthors(feedData, profileData);

        if (!active) return;
        setProfile(profileData);
        setPosts(feedData);
        setAuthors(map);
      } catch (err) {
        if (!active) return;
        if (!handleApiError(err)) {
          setLoadError(
            err instanceof ApiError ? err.message : "Chargement impossible."
          );
        }
      } finally {
        if (active) setLoading(false);
      }
    })();

    return () => {
      active = false;
    };
  }, [handleApiError, resolveAuthors]);

  function authorOf(post) {
    return (
      authors[post.authorId] || { displayName: "Utilisateur", avatarUrl: null }
    );
  }

  return (
    <AppShell right={<FollowedPanel />}>
      <h1 className="main-title">Page d’accueil</h1>

      <Composer
        author={profile}
        onApiError={handleApiError}
        onPublished={(post) => setPosts((prev) => [post, ...prev])}
      />

      <div style={{ marginTop: 16 }}>
        {loading ? (
          <p className="muted">Chargement…</p>
        ) : loadError ? (
          <div className="error">{loadError}</div>
        ) : posts.length === 0 ? (
          <p className="empty">Aucune publication pour le moment.</p>
        ) : (
          posts.map((post) => (
            <PostCard key={post.id} post={post} author={authorOf(post)} />
          ))
        )}
      </div>
    </AppShell>
  );
}
