# Périmètre du MVP Breezy

## Objectif

Développer une version minimale et fonctionnelle de Breezy reposant sur trois services distincts :

1. un service d’authentification ;
2. un service de profil ;
3. un service de publication.

## Parcours utilisateur attendu

1. Un visiteur crée un compte.
2. L’Auth Service crée son compte.
3. L’Auth Service demande au Profile Service de créer automatiquement son profil.
4. L’utilisateur se connecte.
5. L’Auth Service lui retourne un JWT.
6. L’utilisateur consulte son profil.
7. L’utilisateur modifie son nom ou sa biographie.
8. L’utilisateur publie deux messages.
9. Ses publications apparaissent sur son profil, de la plus récente à la plus ancienne.
10. L’utilisateur se déconnecte.

## Fonctionnalités incluses

### Auth Service

* inscription ;
* connexion ;
* déconnexion ;
* récupération de l’identité de l’utilisateur connecté ;
* hachage du mot de passe ;
* génération et validation d’un JWT ;
* création automatique du profil lors de l’inscription.

### Profile Service

* création interne du profil ;
* consultation de son propre profil ;
* modification du nom affiché ;
* modification de la biographie ;
* modification de l’URL de l’avatar.

### Post Service

* création d’un message ;
* limitation du contenu à 280 caractères ;
* récupération des publications de l’utilisateur connecté ;
* classement chronologique décroissant.

### Infrastructure et interface

* trois services Node.js et Express ;
* trois bases PostgreSQL séparées ;
* API Gateway Nginx ;
* orchestration Docker Compose ;
* frontend Next.js minimal ;
* pages d’inscription, de connexion et de profil ;
* tests des parcours critiques.

## Fonctionnalités exclues

Les fonctionnalités suivantes ne seront pas développées dans le MVP :

* likes ;
* commentaires et réponses ;
* abonnements ;
* fil d’actualités global ;
* tags ;
* recherche ;
* notifications ;
* messages privés ;
* upload d’images ;
* upload de vidéos ;
* modération ;
* administration ;
* système de refresh token avancé ;
* broker de messages ;
* Kubernetes ;
* pipeline CI/CD complexe.

## Règles d’architecture

* Chaque service possède sa propre responsabilité.
* Chaque service accède uniquement à sa propre base.
* Aucune clé étrangère ne relie les bases des différents services.
* L’identifiant utilisateur provient du champ `sub` du JWT.
* Le frontend ne choisit jamais librement un `userId` ou un `authorId`.
* Toutes les requêtes du frontend passent par l’API Gateway.
* Les secrets sont placés dans des variables d’environnement.
* Aucun mot de passe en clair n’est enregistré en base.

## Critère de réussite

Le MVP est considéré comme fonctionnel lorsque le scénario complet peut être exécuté sans modification manuelle de la base de données :

```text
Inscription
    ↓
Création automatique du profil
    ↓
Connexion
    ↓
Consultation et modification du profil
    ↓
Publication de deux messages
    ↓
Affichage des publications
    ↓
Déconnexion
```
