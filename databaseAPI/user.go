package databaseAPI

import (
    "database/sql"
    "fmt"
    _ "github.com/mattn/go-sqlite3"
    "golang.org/x/crypto/bcrypt"
)

type User struct {
    IsLoggedIn bool
    Username   string
    MFAEnabled   bool
    MFASecret    string

}



func securePassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}


func GetUser(database *sql.DB, cookie string) string {
    rows, _ := database.Query("SELECT username FROM users WHERE cookie = ?", cookie)
    var username string
    for rows.Next() {
        rows.Scan(&username)
    }
    return username
}


func GetUserInfo(database *sql.DB, submittedEmail string) (string, string, string) {
    var user string
    var email string
    var password string
    rows, _ := database.Query("SELECT username, email, password FROM users WHERE email = ?", submittedEmail)
    for rows.Next() {
        rows.Scan(&user, &email, &password)
    }
    return user, email, password
}


func GetUserByUsername(database *sql.DB, username string) (string, string) {
    var email string
    rows, _ := database.Query("SELECT email FROM users WHERE username = ?", username)
    for rows.Next() {
        rows.Scan(&email)
    }
    return username, email
}


func EditUserProfile(database *sql.DB, username string, newUsername string, email string) bool {
    
    if username != newUsername && !UsernameNotTaken(database, newUsername) {
        return false
    }
    
    statement, err := database.Prepare("UPDATE users SET username = ?, email = ? WHERE username = ?")
    if err != nil {
        return false
    }
    defer statement.Close()
    
    _, err = statement.Exec(newUsername, email, username)
    if err != nil {
        return false
    }
    
    
    statementPosts, err := database.Prepare("UPDATE posts SET username = ? WHERE username = ?")
    if err != nil {
        return false
    }
    defer statementPosts.Close()
    
    _, err = statementPosts.Exec(newUsername, username)
    if err != nil {
        return false
    }
    
    
    statementComments, err := database.Prepare("UPDATE comments SET username = ? WHERE username = ?")
    if err != nil {
        return false
    }
    defer statementComments.Close()
    
    _, err = statementComments.Exec(newUsername, username)
    if err != nil {
        return false
    }
    
    
    statementVotes, err := database.Prepare("UPDATE votes SET username = ? WHERE username = ?")
    if err != nil {
        return false
    }
    defer statementVotes.Close()
    
    _, err = statementVotes.Exec(newUsername, username)
    if err != nil {
        return false
    }
    
    return true
}


func ChangePassword(database *sql.DB, username string, currentPassword string, newPassword string) bool {
    
    var storedPassword string
    err := database.QueryRow("SELECT password FROM users WHERE username = ?", username).Scan(&storedPassword)
    if err != nil {
        return false
    }
    
    
    if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(currentPassword)); err != nil {
        return false
    }
    
    
    hashedPassword, err := securePassword(newPassword)
    if err != nil {
        return false
    }
    
    
    statement, err := database.Prepare("UPDATE users SET password = ? WHERE username = ?")
    if err != nil {
        return false
    }
    defer statement.Close()
    
    _, err = statement.Exec(hashedPassword, username)
    if err != nil {
        return false
    }
    
    return true
}


func GetProfileImage(database *sql.DB, username string) string {
    var imagePath string
    err := database.QueryRow("SELECT profile_image FROM users WHERE username = ?", username).Scan(&imagePath)
    if err != nil || imagePath == "" {
        return "default.png"
    }
    return imagePath
}


func UpdateProfileImage(database *sql.DB, username string, imagePath string) bool {
    statement, err := database.Prepare("UPDATE users SET profile_image = ? WHERE username = ?")
    if err != nil {
        fmt.Println("Erreur préparation SQL:", err)
        return false
    }
    defer statement.Close()
    
    result, err := statement.Exec(imagePath, username)
    if err != nil {
        fmt.Println("Erreur exécution SQL:", err)
        return false
    }
    
    affected, err := result.RowsAffected()
    if err != nil {
        fmt.Println("Erreur vérification lignes affectées:", err)
        return false
    }
    
    if affected == 0 {
        fmt.Println("Aucune ligne modifiée - utilisateur introuvable:", username)
        return false
    }
    
    return true
}


func GetUserStats(database *sql.DB, username string) (int, int, int) {
    var postCount, commentCount, likesReceived int
    
    
    err := database.QueryRow("SELECT COUNT(*) FROM posts WHERE username = ?", username).Scan(&postCount)
    if err != nil {
        postCount = 0
    }
    
    
    err = database.QueryRow("SELECT COUNT(*) FROM comments WHERE username = ?", username).Scan(&commentCount)
    if err != nil {
        commentCount = 0
    }
    
    
    err = database.QueryRow("SELECT COALESCE(SUM(upvotes), 0) FROM posts WHERE username = ?", username).Scan(&likesReceived)
    if err != nil {
        likesReceived = 0
    }
    
    return postCount, commentCount, likesReceived
}


func GetRecentPosts(database *sql.DB, username string, limit int) []Post {
    query := `SELECT id, title, created_at FROM posts WHERE username = ? ORDER BY created_at DESC LIMIT ?`
    rows, err := database.Query(query, username, limit)
    if err != nil {
        fmt.Printf("Erreur requête GetRecentPosts: %v\n", err)
        return []Post{}
    }
    defer rows.Close()
    
    var posts []Post
    for rows.Next() {
        var post Post
        err = rows.Scan(&post.Id, &post.Title, &post.CreatedAt)
        if err != nil {
            fmt.Printf("Erreur scan GetRecentPosts: %v\n", err)
            continue
        }
        posts = append(posts, post)
    }
    
    return posts
}