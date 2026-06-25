# Breezy

Breezy est un MVP de réseau social léger développé avec une architecture distribuée.

## Architecture

L'application repose sur trois services :

- `auth-service` : comptes, connexion et JWT ;
- `profile-service` : informations du profil ;
- `post-service` : création et récupération des publications.

Les requêtes passent par une API Gateway Nginx.

## Technologies prévues

- Node.js
- Express
- Next.js
- PostgreSQL
- Docker Compose
- Nginx
- JWT

## Périmètre

Le périmètre détaillé est disponible dans :

```text
docs/perimetre-mvp.md