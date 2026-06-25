package main

import "daddy/apps/breezy-api/internal/shared"

type userConfig struct {
	Port        string
	PostgresDSN string
	// InternalKey protège l'endpoint interne de synchronisation (POST /internal/users)
	// appelé par l'auth-service au moment du register. Ce n'est pas exposé via le gateway.
	InternalKey string
	NotifURL    string
	// AuthURL pointe vers l'auth-service pour synchroniser l'état de modération
	// (ban/suspension) d'un compte côté authentification. Vide => synchro
	// désactivée (la sanction reste enregistrée mais n'empêche pas le login).
	AuthURL string
	// PostURL pointe vers le post-service, interrogé (endpoint interne) pour
	// classer les suggestions d'abonnement par recouvrement de thèmes. Vide =>
	// suggestions en repli (utilisateurs récents).
	PostURL string
	// Minio héberge les avatars (fichiers image) uploadés via PUT /users/me/avatar.
	Minio minioConfig
}

// minioConfig regroupe l'accès à MinIO et l'URL publique servie au front.
type minioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	PublicURL string
}

func loadUserConfig() userConfig {
	return userConfig{
		Port:        shared.GetEnv("PORT", "3102"),
		PostgresDSN: shared.GetEnvAny([]string{"USER_DATABASE_URL", "DATABASE_URL"}, "postgres://postgres:postgres@localhost:5432/daddy?sslmode=disable"),
		InternalKey: shared.SecretEnv("INTERNAL_API_KEY", "dev-internal-key"),
		NotifURL:    shared.GetEnv("NOTIF_SERVICE_URL", ""),
		AuthURL:     shared.GetEnv("AUTH_SERVICE_URL", ""),
		PostURL:     shared.GetEnv("POST_SERVICE_URL", ""),
		Minio: minioConfig{
			Endpoint:  shared.GetEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: shared.GetEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: shared.GetEnv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    shared.GetEnv("MINIO_BUCKET", "breezy-media"),
			PublicURL: shared.GetEnv("MINIO_PUBLIC_URL", "http://localhost:9000"),
		},
	}
}
