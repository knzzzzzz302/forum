package webAPI

import (
	"FORUM-GO/databaseAPI"
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"net/http"
	"time"
    uuid "github.com/satori/go.uuid"
)


type MFASetupData struct {
	User       User
	QRCodeURL  string
	Secret     string
	Error      string
	Success    string
	MFAEnabled bool
}

type MFAVerifyData struct {
	Username  string
	TempToken string
	Error     string
}

var mfaTempTokens = make(map[string]MFATempToken)

type MFATempToken struct {
	Username  string
	Email     string
	ExpiresAt time.Time
}

func MFASetup(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	cookie, _ := r.Cookie("SESSION")
	username := databaseAPI.GetUser(database, cookie.Value)
	
	mfaEnabled, err := databaseAPI.IsMFAEnabled(database, username)
	if err != nil {
		http.Error(w, "Erreur lors de la vérification MFA", http.StatusInternalServerError)
		return
	}
	
	data := MFASetupData{
		User:       User{IsLoggedIn: true, Username: username},
		MFAEnabled: mfaEnabled,
		Error:      r.URL.Query().Get("error"),
		Success:    r.URL.Query().Get("success"),
	}
	
	if !mfaEnabled && r.URL.Path == "/mfa/setup" && r.Method == "GET" {
		secret, qrURL, err := databaseAPI.GenerateMFASecret(database, username)
		if err != nil {
			http.Error(w, "Erreur lors de la génération du secret MFA", http.StatusInternalServerError)
			return
		}
		
		data.QRCodeURL = qrURL
		data.Secret = secret
	}
	
	t, err := template.ParseFiles("public/HTML/mfa_setup.html")
	if err != nil {
		http.Error(w, "Erreur de template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	t.Execute(w, data)
}

func MFAVerifySetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Erreur de formulaire", http.StatusBadRequest)
		return
	}
	
	cookie, _ := r.Cookie("SESSION")
	username := databaseAPI.GetUser(database, cookie.Value)
	
	code := r.FormValue("code")
	
	valid, err := databaseAPI.VerifyMFACode(database, username, code)
	if err != nil {
		http.Error(w, "Erreur lors de la vérification: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	if !valid {
		http.Redirect(w, r, "/mfa/setup?error=Code+invalide.+Veuillez+réessayer.", http.StatusFound)
		return
	}
	
	http.Redirect(w, r, "/mfa/setup?success=L'authentification+à+deux+facteurs+est+maintenant+activée!", http.StatusFound)
}

func MFADisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	
	cookie, _ := r.Cookie("SESSION")
	username := databaseAPI.GetUser(database, cookie.Value)
	
	err := databaseAPI.DisableMFA(database, username)
	if err != nil {
		http.Error(w, "Erreur lors de la désactivation MFA: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/mfa/setup?success=L'authentification+à+deux+facteurs+a+été+désactivée.", http.StatusFound)
}

func MFALoginCheck(w http.ResponseWriter, r *http.Request, username string, email string) bool {
	mfaEnabled, err := databaseAPI.IsMFAEnabled(database, username)
	if err != nil {
		http.Error(w, "Erreur lors de la vérification MFA", http.StatusInternalServerError)
		return false
	}
	
	if !mfaEnabled {
		return false
	}
	
	token := generateTempToken()
	
	mfaTempTokens[token] = MFATempToken{
		Username:  username,
		Email:     email,
		ExpiresAt: time.Now().Add(10 * time.Minute), 
	}
	
	http.Redirect(w, r, "/mfa/verify?token="+token, http.StatusFound)
	return true
}

func MFAVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	error := r.URL.Query().Get("error")
	
	tempToken, exists := mfaTempTokens[token]
	if !exists || time.Now().After(tempToken.ExpiresAt) {
		http.Redirect(w, r, "/login?err=session_expired", http.StatusFound)
		return
	}
	
	data := MFAVerifyData{
		Username:  tempToken.Username,
		TempToken: token,
		Error:     error,
	}
	
	t, err := template.ParseFiles("public/HTML/mfa_verify.html")
	if err != nil {
		http.Error(w, "Erreur de template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	t.Execute(w, data)
}

func MFAValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Erreur de formulaire", http.StatusBadRequest)
		return
	}
	
	token := r.FormValue("tempToken")
	code := r.FormValue("code")
	
	tempToken, exists := mfaTempTokens[token]
	if !exists || time.Now().After(tempToken.ExpiresAt) {
		http.Redirect(w, r, "/login?err=session_expired", http.StatusFound)
		return
	}
	
	valid, err := databaseAPI.VerifyMFACode(database, tempToken.Username, code)
	if err != nil {
		http.Error(w, "Erreur lors de la vérification: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	if !valid {
		http.Redirect(w, r, "/mfa/verify?token="+token+"&error=Code+invalide.+Veuillez+réessayer.", http.StatusFound)
		return
	}
	
	expiration := time.Now().Add(31 * 24 * time.Hour)
	sessionID := uuid.NewV4().String()
	
	cookie := http.Cookie{
		Name:     "SESSION",
		Value:    sessionID,
		Expires:  expiration,
		Path:     "/",
		HttpOnly: true,
	}
	
	http.SetCookie(w, &cookie)
	
	databaseAPI.UpdateCookie(database, sessionID, expiration, tempToken.Email)
	
	delete(mfaTempTokens, token)
	
	http.Redirect(w, r, "/", http.StatusFound)
}

func generateTempToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}