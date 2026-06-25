package main

import (
	"log"

	"daddy/apps/breezy-api/internal/shared"
)

// main est le point d'entrée du service (connexion DB, migrations, init du
// chiffrement, démarrage HTTP). Code d'infrastructure non couvrable par les
// tests : isolé ici pour être exclu du calcul de couverture (voir .coverignore).
// Le câblage des routes testable vit dans newRouter (main.go).

func main() {
	port := shared.GetEnv("PORT", "3105")
	dsn := shared.GetEnvAny(
		[]string{"MSG_DATABASE_URL", "DATABASE_URL"},
		"postgres://postgres:postgres@localhost:5432/messages?sslmode=disable",
	)

	db, err := shared.ConnectPostgres(dsn)
	if err != nil {
		log.Fatalf("failed to connect message-service database: %v", err)
	}
	if err := shared.AutoMigrate(db, &conversationModel{}, &participantModel{}, &messageModel{}); err != nil {
		log.Fatalf("failed to migrate message-service database: %v", err)
	}

	// Chiffrement au repos du contenu des messages privés.
	if err := initEncryption(shared.SecretEnv("MESSAGE_ENCRYPTION_KEY", "dev-message-encryption-key")); err != nil {
		log.Fatalf("failed to init message encryption: %v", err)
	}

	a := &app{
		db:          db,
		notifURL:    shared.GetEnv("NOTIF_SERVICE_URL", ""),
		internalKey: shared.SecretEnv("INTERNAL_API_KEY", "dev-internal-key"),
	}

	router := newRouter(a)

	log.Printf("message-service listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
