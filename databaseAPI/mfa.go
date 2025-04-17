package databaseAPI

import (
	"database/sql"
	"fmt"      
	"github.com/pquerna/otp/totp"
	"time"     
)

func GenerateMFASecret(database *sql.DB, username string) (string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Forum Sekkay",
		AccountName: username,
	})
	if err != nil {
		return "", "", err
	}
	
	statement, err := database.Prepare("UPDATE users SET mfa_secret = ? WHERE username = ?")
	if err != nil {
		return "", "", err
	}
	_, err = statement.Exec(key.Secret(), username)
	if err != nil {
		return "", "", err
	}
	
	fmt.Printf("MFA Secret généré pour l'utilisateur %s à %s\n", username, time.Now().Format("2006-01-02 15:04:05"))
	
	return key.Secret(), key.URL(), nil
}

func VerifyMFACode(database *sql.DB, username string, code string) (bool, error) {
	var secret string
	err := database.QueryRow("SELECT mfa_secret FROM users WHERE username = ?", username).Scan(&secret)
	if err != nil {
		return false, err
	}
	
	if secret == "" {
		return false, nil
	}
	
	currentTime := time.Now()
	valid := totp.Validate(code, secret)
	
	if valid {
		fmt.Printf("Code MFA valide pour %s à %s\n", username, currentTime.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("Tentative de code MFA invalide pour %s à %s\n", username, currentTime.Format("2006-01-02 15:04:05"))
	}
	
	return valid, nil
}

func IsMFAEnabled(database *sql.DB, username string) (bool, error) {
	var secret string
	err := database.QueryRow("SELECT mfa_secret FROM users WHERE username = ?", username).Scan(&secret)
	if err != nil {
		return false, err
	}
	
	return secret != "", nil
}

func DisableMFA(database *sql.DB, username string) error {
	statement, err := database.Prepare("UPDATE users SET mfa_secret = '' WHERE username = ?")
	if err != nil {
		return err
	}
	_, err = statement.Exec(username)
	
	if err == nil {
		fmt.Printf("MFA désactivé pour l'utilisateur %s à %s\n", username, time.Now().Format("2006-01-02 15:04:05"))
	}
	
	return err
}