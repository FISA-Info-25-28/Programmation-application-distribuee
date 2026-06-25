# API DADdy

Ce dossier contient le backend Go en architecture microservices:
- API Gateway (entrypoint public)
- core-api
- user-service
- product-service

Stack technique:
- Routing HTTP: Gin
- Persistence: GORM
- Base de donnees: PostgreSQL

## Vue d'ensemble de l'architecture

```mermaid
flowchart LR
  C[Client] --> G[api-gateway :3001]
  G -->|/| CORE[core-api :3101]
  G -->|/users + Bearer token| USER[user-service :3102]
  G -->|/products + Bearer token| PROD[product-service :3103]

  USER --> DB[(PostgreSQL)]
  PROD --> DB
  CORE --> DB
```

## Responsabilites des services

- api-gateway
  - Point d'entree unique expose aux clients.
  - Fait du reverse-proxy vers les services internes.
  - Protege `/users` et `/products` via `Authorization: Bearer <token>`.
  - Expose `GET /health`.

- core-api
  - Expose `GET /` (message principal) et `GET /health`.
  - Stocke le message root dans PostgreSQL (table dediee au service).

- user-service
  - Expose `GET /users` et `GET /health`.
  - Lit les utilisateurs depuis PostgreSQL.
  - AutoMigrate + seed idempotent au demarrage.

- product-service
  - Expose `GET /products` et `GET /health`.
  - Lit les produits depuis PostgreSQL.
  - AutoMigrate + seed idempotent au demarrage.

## Organisation du code

```text
apps/api
|-- api-gateway/
|   `-- main.go
|-- core-api/
|   `-- main.go
|-- user-service/
|   `-- main.go
|-- product-service/
|   `-- main.go
|-- internal/
|   `-- shared/
|       `-- shared.go
|-- go.mod
|-- go.sum
`-- package.json
```

`internal/shared/shared.go` mutualise:
- lecture des variables d'environnement
- connexion Postgres (GORM)
- migration automatique (`AutoMigrate`)
- format de reponse health

## Flux d'une requete

1. Le client appelle l'API Gateway (`:3001`).
2. La Gateway applique logging, puis verification token selon la route.
3. La requete est proxyfiee vers le microservice cible.
4. Le microservice execute sa logique Gin + GORM.
5. Le service repond en JSON au client via la Gateway.

## Variables d'environnement

### Gateway
- `PORT` (defaut `3001`)
- `CORE_API_URL` (defaut `http://localhost:3101`)
- `USER_SERVICE_URL` (defaut `http://localhost:3102`)
- `PRODUCT_SERVICE_URL` (defaut `http://localhost:3103`)
- `GATEWAY_TOKEN` (defaut `dev-token`)

### Services metier
Variables communes:
- `PORT` (core: `3101`, user: `3102`, product: `3103`)
- `DATABASE_URL` (fallback global)

Variables specifiques prioritaires:
- core-api: `CORE_DATABASE_URL`
- user-service: `USER_DATABASE_URL`
- product-service: `PRODUCT_DATABASE_URL`

DSN par defaut local utilise dans les services:
- `postgres://postgres:postgres@localhost:5432/daddy?sslmode=disable`

## Demarrage local

Depuis la racine du monorepo:

```bash
npm run dev --workspace=apps/api
```

Ce script lance les 4 services en parallele avec les ports locaux.

## Build

```bash
npm run build --workspace=apps/api
```

Binaires produits dans `apps/api/bin`.

## Endpoints principaux via Gateway

Base URL locale: `http://localhost:3001`

- `GET /`
- `GET /health`
- `GET /users` (Bearer token requis)
- `GET /products` (Bearer token requis)

Exemple:

```bash
curl -H "Authorization: Bearer dev-token" http://localhost:3001/users
curl -H "Authorization: Bearer dev-token" http://localhost:3001/products
```

## Notes d'evolution

- Chaque service possede ses modeles et migrations.
- Les tables sont migrees au boot via GORM `AutoMigrate`.
- En production, vous pouvez remplacer le seed applicatif par un pipeline de migration/seed dedie.
