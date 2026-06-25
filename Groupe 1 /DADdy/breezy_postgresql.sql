-- ============================================================
--  Breezy — Schéma PostgreSQL (version ergonomique)
--  Conventions : PK = id | FK = <entite>_id | timestamps = cree_le / modifie_le
--  POST unifié (breezes + réponses), compteurs maintenus par triggers
-- ============================================================

-- ============================================================
--  TABLES — Cœur (Fx1 à Fx11)
-- ============================================================

CREATE TABLE utilisateur (
    id              BIGSERIAL PRIMARY KEY,
    nom_utilisateur VARCHAR(50)  NOT NULL UNIQUE,
    email           VARCHAR(255) NOT NULL UNIQUE,
    mot_de_passe    VARCHAR(255) NOT NULL,            -- haché (bcrypt) côté API
    bio             TEXT,
    photo_profil    VARCHAR(255),
    role            VARCHAR(20)  NOT NULL DEFAULT 'utilisateur'
                    CHECK (role IN ('utilisateur','moderateur','administrateur')),
    statut          VARCHAR(20)  NOT NULL DEFAULT 'actif'
                    CHECK (statut IN ('actif','suspendu','banni')),
    langue          VARCHAR(5)   NOT NULL DEFAULT 'fr',
    theme           VARCHAR(20)  NOT NULL DEFAULT 'clair',
    nb_abonnes      INT          NOT NULL DEFAULT 0,  -- compteur (trigger)
    nb_abonnements  INT          NOT NULL DEFAULT 0,  -- compteur (trigger)
    cree_le         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    modifie_le      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- POST unifié : breeze si parent_id IS NULL, sinon réponse (Fx7/Fx8)
CREATE TABLE post (
    id              BIGSERIAL PRIMARY KEY,
    contenu         VARCHAR(280) NOT NULL,
    auteur_id       BIGINT NOT NULL REFERENCES utilisateur(id) ON DELETE CASCADE,
    parent_id       BIGINT REFERENCES post(id) ON DELETE CASCADE,
    nb_likes        INT NOT NULL DEFAULT 0,           -- compteur (trigger)
    nb_commentaires INT NOT NULL DEFAULT 0,           -- compteur (trigger)
    cree_le         TIMESTAMPTZ NOT NULL DEFAULT now(),
    modifie_le      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Fil chronologique (Fx5) + messages du profil (Fx11) : on ne liste que les breezes
CREATE INDEX idx_post_feed   ON post (auteur_id, cree_le DESC) WHERE parent_id IS NULL;
-- Réponses d'un post donné (Fx7/Fx8)
CREATE INDEX idx_post_parent ON post (parent_id);

-- JAIME : like (fonctionne sur breeze ET réponse)
CREATE TABLE jaime (
    utilisateur_id BIGINT NOT NULL REFERENCES utilisateur(id) ON DELETE CASCADE,
    post_id        BIGINT NOT NULL REFERENCES post(id) ON DELETE CASCADE,
    cree_le        TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (utilisateur_id, post_id)             -- empêche le double-like
);
CREATE INDEX idx_jaime_post ON jaime (post_id);

-- ABONNEMENT : relation N-M réflexive sur utilisateur
CREATE TABLE abonnement (
    suiveur_id BIGINT NOT NULL REFERENCES utilisateur(id) ON DELETE CASCADE,
    suivi_id   BIGINT NOT NULL REFERENCES utilisateur(id) ON DELETE CASCADE,
    cree_le    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (suiveur_id, suivi_id),
    CHECK (suiveur_id <> suivi_id)                    -- on ne se suit pas soi-même
);
CREATE INDEX idx_abonnement_suivi ON abonnement (suivi_id);

-- ============================================================
--  TABLES — Extensions (Fx12 à Fx23)
-- ============================================================

CREATE TABLE hashtag (
    id      BIGSERIAL PRIMARY KEY,
    libelle VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE post_hashtag (
    post_id    BIGINT NOT NULL REFERENCES post(id) ON DELETE CASCADE,
    hashtag_id BIGINT NOT NULL REFERENCES hashtag(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, hashtag_id)
);
CREATE INDEX idx_post_hashtag_tag ON post_hashtag (hashtag_id);

CREATE TABLE media (
    id         BIGSERIAL PRIMARY KEY,
    post_id    BIGINT NOT NULL REFERENCES post(id) ON DELETE CASCADE,
    type_media VARCHAR(10)  NOT NULL CHECK (type_media IN ('image','video','audio')),
    url        VARCHAR(255) NOT NULL,
    cree_le    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_media_post ON media (post_id);

CREATE TABLE mention (
    post_id        BIGINT NOT NULL REFERENCES post(id) ON DELETE CASCADE,
    utilisateur_id BIGINT NOT NULL REFERENCES utilisateur(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, utilisateur_id)
);

CREATE TABLE notification (
    id              BIGSERIAL PRIMARY KEY,
    destinataire_id BIGINT NOT NULL REFERENCES utilisateur(id) ON DELETE CASCADE,
    type            VARCHAR(20) NOT NULL
                    CHECK (type IN ('like','follow','mention','reponse')),
    donnees         JSONB,                            -- {"post_id":42,"emetteur_id":7}
    lu              BOOLEAN NOT NULL DEFAULT FALSE,
    cree_le         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notif_destinataire ON notification (destinataire_id, cree_le DESC);

CREATE TABLE message_prive (
    id              BIGSERIAL PRIMARY KEY,
    expediteur_id   BIGINT NOT NULL REFERENCES utilisateur(id) ON DELETE CASCADE,
    destinataire_id BIGINT NOT NULL REFERENCES utilisateur(id) ON DELETE CASCADE,
    contenu         TEXT NOT NULL,
    lu              BOOLEAN NOT NULL DEFAULT FALSE,
    cree_le         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_mp_conversation ON message_prive (expediteur_id, destinataire_id, cree_le);

-- Signalement : cible toujours un post (commentaires inclus grâce à la table unifiée)
CREATE TABLE signalement (
    id            BIGSERIAL PRIMARY KEY,
    signaleur_id  BIGINT NOT NULL REFERENCES utilisateur(id) ON DELETE CASCADE,
    post_id       BIGINT NOT NULL REFERENCES post(id) ON DELETE CASCADE,
    moderateur_id BIGINT REFERENCES utilisateur(id) ON DELETE SET NULL,
    motif         VARCHAR(255) NOT NULL,
    statut        VARCHAR(20) NOT NULL DEFAULT 'en_attente'
                  CHECK (statut IN ('en_attente','traite','rejete')),
    cree_le       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sanction (
    id             BIGSERIAL PRIMARY KEY,
    utilisateur_id BIGINT NOT NULL REFERENCES utilisateur(id) ON DELETE CASCADE,
    moderateur_id  BIGINT REFERENCES utilisateur(id) ON DELETE SET NULL,
    type           VARCHAR(20) NOT NULL CHECK (type IN ('suspension','bannissement')),
    motif          VARCHAR(255),
    date_debut     TIMESTAMPTZ NOT NULL DEFAULT now(),
    date_fin       TIMESTAMPTZ,                       -- NULL = définitif
    cree_le        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
--  TRIGGERS — compteurs & horodatage (écrits une fois, oubliés ensuite)
-- ============================================================

-- 1) modifie_le auto sur UPDATE
CREATE OR REPLACE FUNCTION maj_modifie_le() RETURNS TRIGGER AS $$
BEGIN
    NEW.modifie_le = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_utilisateur_modifie BEFORE UPDATE ON utilisateur
    FOR EACH ROW EXECUTE FUNCTION maj_modifie_le();
CREATE TRIGGER trg_post_modifie BEFORE UPDATE ON post
    FOR EACH ROW EXECUTE FUNCTION maj_modifie_le();

-- 2) nb_likes sur post
CREATE OR REPLACE FUNCTION maj_compteur_likes() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        UPDATE post SET nb_likes = nb_likes + 1 WHERE id = NEW.post_id;
    ELSIF (TG_OP = 'DELETE') THEN
        UPDATE post SET nb_likes = nb_likes - 1 WHERE id = OLD.post_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_jaime_compteur AFTER INSERT OR DELETE ON jaime
    FOR EACH ROW EXECUTE FUNCTION maj_compteur_likes();

-- 3) nb_commentaires sur le post parent
CREATE OR REPLACE FUNCTION maj_compteur_commentaires() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT' AND NEW.parent_id IS NOT NULL) THEN
        UPDATE post SET nb_commentaires = nb_commentaires + 1 WHERE id = NEW.parent_id;
    ELSIF (TG_OP = 'DELETE' AND OLD.parent_id IS NOT NULL) THEN
        UPDATE post SET nb_commentaires = nb_commentaires - 1 WHERE id = OLD.parent_id;
    ELSIF (TG_OP = 'UPDATE' AND NEW.parent_id IS DISTINCT FROM OLD.parent_id) THEN
        IF (OLD.parent_id IS NOT NULL) THEN
            UPDATE post SET nb_commentaires = nb_commentaires - 1 WHERE id = OLD.parent_id;
        END IF;
        IF (NEW.parent_id IS NOT NULL) THEN
            UPDATE post SET nb_commentaires = nb_commentaires + 1 WHERE id = NEW.parent_id;
        END IF;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_post_compteur_commentaires AFTER INSERT OR DELETE OR UPDATE OF parent_id ON post
    FOR EACH ROW EXECUTE FUNCTION maj_compteur_commentaires();

-- 4) nb_abonnes / nb_abonnements sur utilisateur
CREATE OR REPLACE FUNCTION maj_compteur_abonnements() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        UPDATE utilisateur SET nb_abonnes    = nb_abonnes + 1    WHERE id = NEW.suivi_id;
        UPDATE utilisateur SET nb_abonnements = nb_abonnements + 1 WHERE id = NEW.suiveur_id;
    ELSIF (TG_OP = 'DELETE') THEN
        UPDATE utilisateur SET nb_abonnes    = nb_abonnes - 1    WHERE id = OLD.suivi_id;
        UPDATE utilisateur SET nb_abonnements = nb_abonnements - 1 WHERE id = OLD.suiveur_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_abonnement_compteur AFTER INSERT OR DELETE ON abonnement
    FOR EACH ROW EXECUTE FUNCTION maj_compteur_abonnements();

-- ============================================================
--  EXEMPLES — requêtes qui deviennent triviales
-- ============================================================

-- Fil chronologique d'un utilisateur (Fx5)
--   SELECT p.* FROM post p
--   JOIN abonnement a ON a.suivi_id = p.auteur_id
--   WHERE a.suiveur_id = :moi AND p.parent_id IS NULL
--   ORDER BY p.cree_le DESC LIMIT 20;

-- Messages du profil (Fx4/Fx11) — compteurs déjà inclus dans la ligne
--   SELECT * FROM post WHERE auteur_id = :user AND parent_id IS NULL ORDER BY cree_le DESC;

-- Réponses sous un post (Fx7/Fx8)
--   SELECT * FROM post WHERE parent_id = :post_id ORDER BY cree_le ASC;

-- Centre de notifications (Fx14/15/16)
--   SELECT * FROM notification WHERE destinataire_id = :moi ORDER BY cree_le DESC;
