package webAPI

import (
	"FORUM-GO/databaseAPI"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	uuid "github.com/satori/go.uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/github"
)

var googleOauthConfig = &oauth2.Config{
	ClientID:     "786149339952-g4vqhj3rficg4a0379i46pehddgut82l.apps.googleusercontent.com",
	ClientSecret: "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",
	RedirectURL:  "https://localhost:3030/auth/google/callback",
	Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
	Endpoint:     google.Endpoint,
}

var githubOauthConfig = &oauth2.Config{
	ClientID:     "Ov23lilyZ5reXy1ET3qk",     
	ClientSecret: "e0fd64f5b1adeea7f5678bf951f78d16f525aea6", 
	RedirectURL:  "https://localhost:3030/auth/github/callback",
	Scopes:       []string{"user:email"},
	Endpoint:     github.Endpoint,
}


func generateStateOauthCookie(w http.ResponseWriter) string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{
		Name:     "oauthstate",
		Value:    state,
		Expires:  time.Now().Add(time.Hour),
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	return state
}

func generateGitHubStateOauthCookie(w http.ResponseWriter) string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{
		Name:     "githubstate",
		Value:    state,
		Expires:  time.Now().Add(time.Hour),
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	return state
}

func GoogleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateStateOauthCookie(w)
	url := googleOauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func GitHubLogin(w http.ResponseWriter, r *http.Request) {
	state := generateGitHubStateOauthCookie(w)
	url := githubOauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type GitHubUserInfo struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func GoogleCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, err := r.Cookie("oauthstate")
	if err != nil {
		http.Error(w, "État de cookie invalide", http.StatusBadRequest)
		return
	}

	if r.FormValue("state") != oauthState.Value {
		http.Error(w, "État invalide", http.StatusBadRequest)
		return
	}

	token, err := googleOauthConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		http.Error(w, "Échec d'échange de code", http.StatusInternalServerError)
		return
	}

	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		http.Error(w, "Échec de récupération des infos utilisateur", http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		http.Error(w, "Échec de lecture des données utilisateur", http.StatusInternalServerError)
		return
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(data, &userInfo); err != nil {
		http.Error(w, "Échec de désérialisation JSON", http.StatusInternalServerError)
		return
	}

	userExists := !databaseAPI.EmailNotTaken(database, userInfo.Email)
	
	if !userExists {
		password := generateRandomPassword(16)
		expiration := time.Now().Add(31 * 24 * time.Hour)
		value := uuid.NewV4().String()
		
		databaseAPI.AddUser(database, userInfo.Name, userInfo.Email, password, value, expiration.Format("2006-01-02 15:04:05"))
		fmt.Printf("Nouvel utilisateur créé via Google: %s (%s)\n", userInfo.Name, userInfo.Email)
	} else {
		username, _, _ := databaseAPI.GetUserInfo(database, userInfo.Email)
		expiration := time.Now().Add(31 * 24 * time.Hour)
		value := uuid.NewV4().String()
		
		databaseAPI.UpdateCookie(database, value, expiration, userInfo.Email)
		fmt.Printf("Utilisateur connecté via Google: %s (%s)\n", username, userInfo.Email)
	}

	expiration := time.Now().Add(31 * 24 * time.Hour)
	value := uuid.NewV4().String()
	cookie := http.Cookie{Name: "SESSION", Value: value, Expires: expiration, Path: "/"}
	http.SetCookie(w, &cookie)
	
	databaseAPI.UpdateCookie(database, value, expiration, userInfo.Email)
	
	http.Redirect(w, r, "/", http.StatusFound)
}

func GitHubCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, err := r.Cookie("githubstate")
	if err != nil {
		http.Error(w, "État de cookie invalide", http.StatusBadRequest)
		return
	}

	if r.FormValue("state") != oauthState.Value {
		http.Error(w, "État invalide", http.StatusBadRequest)
		return
	}

	token, err := githubOauthConfig.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		http.Error(w, "Échec d'échange de code", http.StatusInternalServerError)
		return
	}

	client := githubOauthConfig.Client(context.Background(), token)
	response, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Error(w, "Échec de récupération des infos utilisateur", http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		http.Error(w, "Échec de lecture des données utilisateur", http.StatusInternalServerError)
		return
	}

	var userInfo GitHubUserInfo
	if err := json.Unmarshal(data, &userInfo); err != nil {
		http.Error(w, "Échec de désérialisation JSON", http.StatusInternalServerError)
		return
	}

	emailResponse, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		http.Error(w, "Échec de récupération des emails", http.StatusInternalServerError)
		return
	}
	defer emailResponse.Body.Close()

	emailData, err := ioutil.ReadAll(emailResponse.Body)
	if err != nil {
		http.Error(w, "Échec de lecture des emails", http.StatusInternalServerError)
		return
	}

	var emails []struct {
		Email    string `json:"email"`
		Verified bool   `json:"verified"`
		Primary  bool   `json:"primary"`
	}
	if err := json.Unmarshal(emailData, &emails); err != nil {
		http.Error(w, "Échec de désérialisation des emails", http.StatusInternalServerError)
		return
	}

	var primaryEmail string
	for _, email := range emails {
		if email.Verified && email.Primary {
			primaryEmail = email.Email
			break
		}
	}

	userExists := !databaseAPI.EmailNotTaken(database, primaryEmail)
	
	if !userExists {
		password := generateRandomPassword(16)
		expiration := time.Now().Add(31 * 24 * time.Hour)
		value := uuid.NewV4().String()
		
		databaseAPI.AddUser(database, userInfo.Login, primaryEmail, password, value, expiration.Format("2006-01-02 15:04:05"))
		fmt.Printf("Nouvel utilisateur créé via GitHub: %s (%s)\n", userInfo.Login, primaryEmail)
	} else {
		username, _, _ := databaseAPI.GetUserInfo(database, primaryEmail)
		expiration := time.Now().Add(31 * 24 * time.Hour)
		value := uuid.NewV4().String()
		
		databaseAPI.UpdateCookie(database, value, expiration, primaryEmail)
		fmt.Printf("Utilisateur connecté via GitHub: %s (%s)\n", username, primaryEmail)
	}

	expiration := time.Now().Add(31 * 24 * time.Hour)
	value := uuid.NewV4().String()
	cookie := http.Cookie{Name: "SESSION", Value: value, Expires: expiration, Path: "/"}
	http.SetCookie(w, &cookie)
	
	databaseAPI.UpdateCookie(database, value, expiration, primaryEmail)
	
	http.Redirect(w, r, "/", http.StatusFound)
}

func generateRandomPassword(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}