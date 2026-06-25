# Breezy

Breezy est un MVP de réseau social distribué permettant de créer un compte, gérer son profil et publier des messages courts.

## Architecture

L’application comprend :

* un service d’authentification
* un service de profil
* un service de publication
* un frontend Next.js
* une API Gateway Nginx
* une base PostgreSQL

## Lancement du projet

### Prérequis

* Git
* Docker Desktop

### Installation

Cloner le dépôt puis ouvrir un terminal à sa racine :

```powershell
git clone <URL_DU_DEPOT>
cd breezy
```

Créer le fichier `.env` à partir du modèle :

```powershell
Copy-Item .env.example .env
```

Construire et démarrer l’application :

```powershell
docker compose up --build
```

L’application est ensuite accessible à l’adresse :

http://localhost:3000

## Arrêt du projet

```powershell
docker compose down
```

Pour supprimer également les données enregistrées :

```powershell
docker compose down -v
```
