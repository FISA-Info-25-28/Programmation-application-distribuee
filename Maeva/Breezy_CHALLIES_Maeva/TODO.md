# Suivi du MVP Breezy

## 1. Préparation et périmètre

* [x] Initialiser le dépôt Git
* [x] Créer la structure du projet
* [x] Définir le périmètre du MVP
* [x] Définir le scénario de démonstration
* [x] Identifier les variables d’environnement
* [x] Effectuer le commit initial

## 2. Infrastructure Docker

* [x] Créer le fichier Docker Compose
* [x] Configurer PostgreSQL
* [x] Créer les bases `auth_db`, `profile_db` et `post_db`
* [x] Créer le conteneur Auth Service
* [x] Créer le conteneur Profile Service
* [x] Créer le conteneur Post Service
* [x] Créer le conteneur Frontend
* [x] Créer le conteneur Gateway

## 3. Auth Service

* [x] Créer la route `/health`
* [x] Créer la table `users`
* [x] Développer l’inscription
* [x] Hacher les mots de passe
* [x] Développer la connexion
* [x] Générer le JWT
* [x] Créer le middleware JWT
* [x] Développer `/api/auth/me`
* [x] Développer la déconnexion
* [x] Appeler le Profile Service à l’inscription

## 4. Profile Service

* [x] Créer la route `/health`
* [x] Créer la table `profiles`
* [x] Créer la route interne de création
* [x] Vérifier la clé API interne
* [x] Créer le middleware JWT
* [x] Consulter son profil
* [x] Modifier son profil
* [x] Valider les longueurs des champs

## 5. Post Service

* [x] Créer la route `/health`
* [x] Créer la table `posts`
* [x] Créer le middleware JWT
* [x] Publier un message
* [x] Limiter les messages à 280 caractères
* [x] Récupérer ses publications
* [x] Trier les publications par date décroissante

## 6. API Gateway

* [x] Router `/api/auth`
* [x] Router `/api/profile`
* [x] Router `/api/posts`
* [x] Vérifier les trois routes `/health`

## 7. Frontend

* [x] Créer la page d’inscription
* [x] Créer la page de connexion
* [x] Gérer le JWT
* [x] Créer la page de profil
* [x] Modifier le profil
* [x] Créer un post
* [x] Afficher les publications
* [x] Ajouter la déconnexion
* [x] Gérer les erreurs principales

## 8. Tests

* [ ] Tester une inscription valide
* [ ] Tester un email déjà utilisé
* [ ] Tester une mauvaise connexion
* [ ] Tester les routes sans JWT
* [ ] Tester la modification du profil
* [ ] Tester une biographie trop longue
* [ ] Tester un post valide
* [ ] Tester un post vide
* [ ] Tester un post supérieur à 280 caractères

## 9. Documentation

* [ ] Documenter l’installation
* [ ] Documenter le lancement Docker
* [ ] Documenter les routes
* [ ] Documenter le scénario de démonstration
* [ ] Documenter les limitations du MVP
