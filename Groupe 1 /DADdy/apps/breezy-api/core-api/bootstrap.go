package main

import (
	"log"

	"daddy/apps/breezy-api/internal/models"
	"daddy/apps/breezy-api/internal/shared"
)

// main est le point d'entrée du service (connexion DB, migrations, démarrage du
// serveur HTTP). Code d'infrastructure non couvrable par les tests : isolé ici
// pour être exclu du calcul de couverture (voir .coverignore). Le câblage des
// routes testable vit dans newRouter (main.go).

func main() {
	port := shared.GetEnv("PORT", "3101")
	postgresDSN := shared.GetEnvAny([]string{"CORE_DATABASE_URL", "DATABASE_URL"}, "postgres://postgres:postgres@localhost:5432/daddy?sslmode=disable")

	db, err := shared.ConnectPostgres(postgresDSN)
	if err != nil {
		log.Fatalf("failed to connect core-api database: %v", err)
	}

	if err := shared.AutoMigrate(db,
		&models.User{},
		&models.Post{},
		&models.PostTag{},
		&models.Comment{},
		&models.Like{},
	); err != nil {
		log.Fatalf("failed to migrate core-api database: %v", err)
	}

	srv := &server{
		db:             db,
		internalSecret: shared.SecretEnv("INTERNAL_API_KEY", "dev-internal-key"),
	}

	router := newRouter(srv)

	log.Printf("core-api listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
