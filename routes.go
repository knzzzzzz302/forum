package routes

import (
	"forum/handlers"
	"net/http"
)

// SetupRoutes initialise les routes
func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/login", handlers.LoginHandler)
	return mux
}
