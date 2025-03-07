package main

import (
	"forum/config"
	"forum/routes"
	"log"
	"net/http"
)

func main() {
	// Connexion à la base de données
	config.ConnectDB()

	// Chargement des routes
	mux := routes.SetupRoutes()

	// Démarrer le serveur
	log.Println("Serveur démarré sur http://localhost:8080")
	http.ListenAndServe(":8080", mux)
}
