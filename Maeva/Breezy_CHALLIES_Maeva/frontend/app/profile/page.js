"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import AppShell from "../../components/AppShell";
import Avatar from "../../components/Avatar";
import PostCard from "../../components/PostCard";
import {
  profileApi,
  postsApi,
  getToken,
  clearToken,
  ApiError
} from "../../services/api";
import { handleFromName } from "../../lib/format";

const MAX_BIO_LENGTH = 160;

export default function ProfilePage() {
  const router = useRouter();

  const [profile, setProfile] = useState(null);
  const [posts, setPosts] = useState([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(null);
  const [editing, setEditing] = useState(false);

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

  useEffect(() => {
    if (!getToken()) return;

    let active = true;

    (async () => {
      try {
        const [profileData, postsData] = await Promise.all([
          profileApi.me(),
          postsApi.list()
        ]);
        if (!active) return;
        setProfile(profileData);
        setPosts(postsData);
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

    return () => { active = false; };
  }, [handleApiError]);

  if (loading) {
    return <AppShell><p className="muted">Chargement…</p></AppShell>;
  }

  if (loadError) {
    return <AppShell><div className="error">{loadError}</div></AppShell>;
  }

  if (editing) {
    return (
      <AppShell>
        <ProfileEditForm
          profile={profile}
          onUpdated={(p) => { setProfile(p); setEditing(false); }}
          onCancel={() => setEditing(false)}
          onApiError={handleApiError}
        />
      </AppShell>
    );
  }

  return (
    <AppShell>
      <div className="profile-hero">
        <div className="profile-hero-top">
          <div className="profile-avatar-ring">
            <Avatar
              avatarUrl={profile.avatarUrl}
              displayName={profile.displayName}
              size="lg"
            />
          </div>
          <button
            className="btn-outline profile-edit-btn"
            onClick={() => setEditing(true)}
          >
            Modifier le profil
          </button>
        </div>

        <h1 className="profile-hero-name">{profile.displayName}</h1>
        <p className="profile-hero-handle">{handleFromName(profile.displayName)}</p>

        {profile.bio && (
          <p className="profile-hero-bio">{profile.bio}</p>
        )}

        <p className="profile-hero-count">
          {posts.length === 0
            ? "Aucune publication"
            : `${posts.length} publication${posts.length > 1 ? "s" : ""}`}
        </p>
      </div>

      <div className="profile-feed-divider" />

      {posts.length === 0 ? (
        <div className="profile-empty">
          <p className="profile-empty-title">Aucun post pour le moment</p>
          <p className="profile-empty-sub">Tes prochains posts apparaîtront ici.</p>
        </div>
      ) : (
        posts.map((post) => (
          <PostCard key={post.id} post={post} author={profile} />
        ))
      )}
    </AppShell>
  );
}

/* ------------------------------------------------------------------ */
/* Formulaire de modification du profil                                */
/* ------------------------------------------------------------------ */

function ProfileEditForm({ profile, onUpdated, onCancel, onApiError }) {
  const [displayName, setDisplayName] = useState(profile.displayName);
  const [bio, setBio] = useState(profile.bio || "");
  const [avatarUrl, setAvatarUrl] = useState(profile.avatarUrl || "");
  const [error, setError] = useState(null);
  const [saving, setSaving] = useState(false);

  async function handleSubmit(e) {
    e.preventDefault();
    setError(null);

    if (!displayName.trim()) {
      setError("Le nom d'affichage est obligatoire.");
      return;
    }

    setSaving(true);

    try {
      const result = await profileApi.update({
        displayName: displayName.trim(),
        bio: bio.trim(),
        avatarUrl: avatarUrl.trim()
      });
      onUpdated(result.profile);
    } catch (err) {
      if (onApiError(err)) return;
      setError(err instanceof ApiError ? err.message : "Mise à jour impossible.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="card">
      <h2 style={{ fontSize: "1.2rem", marginBottom: 20 }}>Modifier le profil</h2>

      {error && <div className="error">{error}</div>}

      <form onSubmit={handleSubmit} noValidate>
        <div className="field">
          <label htmlFor="displayName">Nom d&apos;affichage</label>
          <input
            id="displayName"
            type="text"
            maxLength={50}
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
        </div>

        <div className="field">
          <label htmlFor="bio">Biographie</label>
          <textarea
            id="bio"
            maxLength={MAX_BIO_LENGTH}
            value={bio}
            onChange={(e) => setBio(e.target.value)}
          />
          <span className="counter">{bio.length} / {MAX_BIO_LENGTH}</span>
        </div>

        <div className="field">
          <label htmlFor="avatarUrl">URL de l&apos;avatar</label>
          <input
            id="avatarUrl"
            type="url"
            maxLength={500}
            placeholder="https://exemple.com/avatar.png"
            value={avatarUrl}
            onChange={(e) => setAvatarUrl(e.target.value)}
          />
        </div>

        <div style={{ display: "flex", gap: 12, justifyContent: "flex-end" }}>
          <button type="button" className="btn-ghost" onClick={onCancel} disabled={saving}>
            Annuler
          </button>
          <button type="submit" className="btn-dark" disabled={saving}>
            {saving ? "Enregistrement…" : "Enregistrer"}
          </button>
        </div>
      </form>
    </section>
  );
}
