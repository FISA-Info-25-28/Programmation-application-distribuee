# DADdy

Monorepo full-stack du projet **Breezy** (reseau social type microblogging).

## Architecture

- Frontend web: Vite + React dans `apps/breezy-web`
- Application mobile: Capacitor (iOS/Android) dans `apps/breezy-mobile`
- Code partage: `packages/breezy-shared` (`@breezy/shared`)
- Backend: Go microservices dans `apps/breezy-api`

| Service | Port | Role |
|---|---|---|
| `api-gateway` | 3001 | Point d'entree unique: verif JWT, rate-limiting (Valkey), CORS, routage |
| `core-api` | 3101 | Endpoints transverses / sante |
| `user-service` | 3102 | Profils, follows, demandes de suivi, avatars/bannieres |
| `post-service` | 3103 | Posts, feed, likes, commentaires, hashtags |
| `auth-service` | 3104 | Inscription/connexion, JWT, OAuth, verif email, reset mdp, MFA |
| `message-service` | 3105 | Messages prives (chiffres au repos) |
| `notification-service` | 3106 | Notifications |

Infra:
- PostgreSQL (une base par service): core `5433`, user `5434`, auth `5435`, post `5436`, message `5437`, notif `5438`
- MinIO: stockage des medias/avatars (`9000` API, `9001` console)
- Valkey: cache / rate-limiting du gateway
- Swagger UI (spec API): `8081`
- pgAdmin: `8080`
- Mailpit (dev uniquement): capture des mails, UI sur `8025`

## Prerequisites

- Node.js 20+
- npm 10+
- Go 1.25+
- Docker Compose ou Podman Compose (recommande pour le dev env)

## Installation

```bash
npm install
```

## Demarrer l'environnement de dev

### Option A (recommandee): tout lancer avec Compose (Docker ou Podman)

1. Creer le fichier d'env:

```bash
cp .env.example .env
```

2. Lancer la stack:

```bash
npm run compose -- up --build
```

Services exposes:
- Web: `http://localhost:5173`
- API Gateway: `http://localhost:3001`
- Swagger UI: `http://localhost:8081`
- pgAdmin: `http://localhost:8080`
- MinIO console: `http://localhost:9001`
- Mailpit (mails de dev): `http://localhost:8025`

### Option B: lancer en local via npm workspaces

Cette option demarre automatiquement les bases Postgres core/user/auth locales (Docker/Podman):
- `core-db` sur `localhost:5433`
- `user-db` sur `localhost:5434`
- `auth-db` sur `localhost:5435`

Depuis la racine:

```bash
npm run dev
```

Cela lance:
- Web (Vite)
- API (gateway + tous les microservices)

> Note: les services `post`, `message` et `notification` necessitent leurs bases
> (`post-db`, `msg-db`, `notif-db`), MinIO et le cache. Pour une stack complete,
> preferer l'Option A.

### Seed des donnees de demo

```bash
npm run seed
```

### Application mobile (Capacitor)

Build du web + sync natif, puis ouverture du projet iOS/Android:

```bash
npm run cap:sync --workspace=apps/breezy-mobile
npm run cap:ios --workspace=apps/breezy-mobile
npm run cap:android --workspace=apps/breezy-mobile
```

## Scripts utiles

Racine:

```bash
npm run dev
npm run build
npm run lint        # eslint (web) + go vet (api)
npm run seed
npm run build:docs  # Swagger API + TypeDoc web + page d'accueil docs
```

Web:

```bash
npm run dev --workspace=apps/breezy-web
npm run build --workspace=apps/breezy-web
npm run lint --workspace=apps/breezy-web
```

API:

```bash
npm run dev --workspace=apps/breezy-api
npm run build --workspace=apps/breezy-api
npm run start --workspace=apps/breezy-api
```

## Variables d'environnement

Voir `.env.example` pour la liste complete:
- Bases de donnees: core, user, auth, post, message, notif
- JWT: `JWT_SECRET`, `JWT_ISSUER`, rotation de cles, TTL access/refresh
- `INTERNAL_API_KEY`: appels internes entre services
- MinIO: identifiants, bucket, URL publique
- Valkey: `VALKEY_PASSWORD`, limites de rate-limiting
- Email (SMTP) + verification / reset de mot de passe
- OAuth2: Google et GitHub
- `MESSAGE_ENCRYPTION_KEY`: chiffrement des messages prives

## Endpoints principaux (via gateway)

Base URL locale: `http://localhost:3001`

- `GET /health`
- **Auth** `/auth/*`: `register`, `login`, `logout`, `refresh`, `me`, `verify-email`,
  `resend-verification`, `request-password-reset`, `reset-password`, `change-password`,
  `mfa/setup|confirm|verify`, `oauth/google`, `oauth/github`
- **Users** `/users/*`: liste, `{id}`, `follow`, `followers`, `following`, `me`,
  `me/avatar`, `me/banner`, `me/follow-requests`, `suggestions`
- **Posts** `/posts/*`: `posts`, `{id}`, `{id}/like`, `{id}/comments`, `for-you`,
  `feed`, `hashtags/trending`
- **Notifications**: `GET /notifications`
- **Messages**: messagerie privee via `message-service`

Routes protegees: header `Authorization: Bearer <JWT>`.

## Deploiement

- Conteneurs: voir `compose.yaml` (et `compose.override.yaml` pour le dev).
- Kubernetes: manifests Kustomize + ArgoCD dans `.kubernetes/` (cf. `.kubernetes/README.md`).
- Provisioning: playbooks Ansible dans `ansible/` (cf. `ansible/README.md`).
- CI/CD: workflows GitHub Actions dans `.github/workflows/`.

## Notes

- Le repo utilise npm workspaces (`apps/*`, `packages/*`).
- La documentation est generee automatiquement (Swagger UI pour API, TypeDoc pour Web).
