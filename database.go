package config

import (
	"database/sql"
	"fmt"
	"log"
    "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func ConnectDB() {
	var err error
	DB, err = sql.Open("sqlite3", "forum.db")
	if err != nil {
		log.Fatal("Erreur de connexion à la base de données:", err)
	}

	// Création de la table users si elle n'existe pas
	createTable := `CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password TEXT
	);`
	_, err = DB.Exec(createTable)
	if err != nil {s
		log.Fatal("Erreur lors de la création des tables:", err)
	}

	fmt.Println("Base de données connectée et initialisée !")
}
