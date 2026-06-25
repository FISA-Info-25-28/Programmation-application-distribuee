package main

import (
	"log"
)

func main() {
	cfg := loadGatewayConfig()
	router := newGatewayRouter(cfg)
	log.Printf("api-gateway listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
