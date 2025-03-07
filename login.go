	package handlers

	import (
		"crypto/sha256"
		"encoding/hex"
		"forum/config"
		"forum/models"
		"net/http"
		"text/template"
	)

	// LoginHandler gère l'authentification des utilisateurs
	func LoginHandler(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl, err := template.ParseFiles("templates/login.html")
			if err != nil {
				http.Error(w, "Erreur de chargement du template", http.StatusInternalServerError)
				return
			}
			tmpl.Execute(w, nil)
			return
		}

		if r.Method == http.MethodPost {
			r.ParseForm()
			username := r.FormValue("username")
			password := r.FormValue("password")

			var user models.User
			err := config.DB.QueryRow("SELECT id, password FROM users WHERE username = ?", username).Scan(&user.ID, &user.Password)
			if err != nil {
				http.Error(w, "Utilisateur non trouvé", http.StatusUnauthorized)
				return
			}

			// Vérifier le mot de passe hashé
			hashedPassword := hashPassword(password)
			if hashedPassword != user.Password {
				http.Error(w, "Mot de passe incorrect", http.StatusUnauthorized)
				return
			}

			// Création de session (simplifiée)
			http.SetCookie(w, &http.Cookie{
				Name:  "session",
				Value: username,
				Path:  "/",
			})

			http.Redirect(w, r, "/home", http.StatusSeeOther)
		}
	}

	func hashPassword(password string) string {
		hash := sha256.Sum256([]byte(password))
		return hex.EncodeToString(hash[:])
	}
