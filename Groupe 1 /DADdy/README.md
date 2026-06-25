# DADdy

Monorepo full-stack du projet **Breezy** avec:
- Frontend web: Vite + React dans `apps/breezy-web`
- Application mobile: Capacitor (iOS/Android) dans `apps/breezy-mobile`
- Backend: Go microservices dans `apps/breezy-api`
	- `api-gateway`
	- `core-api`
	- `user-service`
	- `auth-service`
- Code partage: `packages/breezy-shared` (`@breezy/shared`)

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
- Swagger UI (spec brute): `http://localhost:8081`

### Option B: lancer en local via npm workspaces

Cette option demarre automatiquement les bases Postgres locales (Docker/Podman):
- `core-db` sur `localhost:5433`
- `user-db` sur `localhost:5434`
- `auth-db` sur `localhost:5435`

Depuis la racine:

```bash
npm run dev
```

Cela lance:
- Web (Vite)
- API (gateway + core-api + user-service + auth-service)

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
npm run lint
npm run build:docs
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
- Databases: core, user, auth
- JWT: `JWT_SECRET`, `JWT_ISSUER`

## Endpoints principaux (via gateway)

Base URL locale: `http://localhost:3001`

- `GET /health`
- `POST /auth/register`
- `POST /auth/login`
- `GET /auth/me` (Bearer JWT)
- `GET /users` (Bearer JWT)

## Deploiement

- Conteneurs: voir `compose.yaml` (et `compose.override.yaml` pour le dev).
- Kubernetes: manifests Kustomize + ArgoCD dans `.kubernetes/` (cf. `.kubernetes/README.md`).
- Provisioning: playbooks Ansible dans `ansible/` (cf. `ansible/README.md`).
- CI/CD: workflows GitHub Actions dans `.github/workflows/`.

## Notes

- Le repo utilise npm workspaces (`apps/*`, `packages/*`).
- La documentation est generee automatiquement (Swagger UI pour API, TypeDoc pour Web).