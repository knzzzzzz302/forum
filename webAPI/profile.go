package webAPI

import (
    "FORUM-GO/databaseAPI"
    "fmt"
    "html/template"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"
)

type ProfilePage struct {
    User          User
    Username      string
    Email         string
    ProfileImage  string
    PostCount     int
    CommentCount  int
    LikesReceived int
    RecentPosts   []databaseAPI.Post
    Message       string
    MFAEnabled    bool 
}

func DisplayProfile(w http.ResponseWriter, r *http.Request) {
    if !isLoggedIn(r) {
        http.Redirect(w, r, "/login", http.StatusFound)
        return
    }

    cookie, _ := r.Cookie("SESSION")
    username := databaseAPI.GetUser(database, cookie.Value)
    
    username, email := databaseAPI.GetUserByUsername(database, username)
    profileImage := databaseAPI.GetProfileImage(database, username)
    
    postCount, commentCount, likesReceived := databaseAPI.GetUserStats(database, username)
    recentPosts := databaseAPI.GetRecentPosts(database, username, 5)
    
    message := r.URL.Query().Get("msg")
    
    mfaEnabled, _ := databaseAPI.IsMFAEnabled(database, username)
    
    payload := ProfilePage{
        User:          User{IsLoggedIn: true, Username: username},
        Username:      username,
        Email:         email,
        ProfileImage:  profileImage,
        PostCount:     postCount,
        CommentCount:  commentCount,
        LikesReceived: likesReceived,
        RecentPosts:   recentPosts,
        Message:       message,
        MFAEnabled:    mfaEnabled, 
    }
    
    t, err := template.ParseFiles("public/HTML/profile.html")
    if err != nil {
        http.Error(w, "Erreur lors du chargement de la page: "+err.Error(), http.StatusInternalServerError)
        return
    }
    
    err = t.Execute(w, payload)
    if err != nil {
        http.Error(w, "Erreur lors de l'affichage de la page: "+err.Error(), http.StatusInternalServerError)
    }
}
func EditProfileHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
        return
    }

    if !isLoggedIn(r) {
        http.Redirect(w, r, "/login", http.StatusFound)
        return
    }

    if err := r.ParseForm(); err != nil {
        http.Error(w, fmt.Sprintf("Erreur de ParseForm(): %v", err), http.StatusBadRequest)
        return
    }

    cookie, _ := r.Cookie("SESSION")
    username := databaseAPI.GetUser(database, cookie.Value)
    
    newUsername := r.FormValue("username")
    email := r.FormValue("email")
    
    if newUsername == "" || email == "" {
        http.Redirect(w, r, "/profile?msg=empty_fields", http.StatusFound)
        return
    }
    
    success := databaseAPI.EditUserProfile(database, username, newUsername, email)
    if !success {
        http.Redirect(w, r, "/profile?msg=update_failed", http.StatusFound)
        return
    }
    
    http.Redirect(w, r, "/profile?msg=profile_updated", http.StatusFound)
}

func ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
        return
    }

    if !isLoggedIn(r) {
        http.Redirect(w, r, "/login", http.StatusFound)
        return
    }

    if err := r.ParseForm(); err != nil {
        http.Error(w, fmt.Sprintf("Erreur de ParseForm(): %v", err), http.StatusBadRequest)
        return
    }

    cookie, _ := r.Cookie("SESSION")
    username := databaseAPI.GetUser(database, cookie.Value)
    
    currentPassword := r.FormValue("current_password")
    newPassword := r.FormValue("new_password")
    confirmPassword := r.FormValue("confirm_password")
    
    if currentPassword == "" || newPassword == "" || confirmPassword == "" {
        http.Redirect(w, r, "/profile?msg=empty_password_fields", http.StatusFound)
        return
    }
    
    if newPassword != confirmPassword {
        http.Redirect(w, r, "/profile?msg=passwords_dont_match", http.StatusFound)
        return
    }
    
    success := databaseAPI.ChangePassword(database, username, currentPassword, newPassword)
    if !success {
        http.Redirect(w, r, "/profile?msg=password_change_failed", http.StatusFound)
        return
    }
    
    http.Redirect(w, r, "/profile?msg=password_changed", http.StatusFound)
}

func UploadProfileImageHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
        return
    }
    
    if !isLoggedIn(r) {
        http.Redirect(w, r, "/login", http.StatusFound)
        return
    }
    
    err := r.ParseMultipartForm(10 << 20) 
    if err != nil {
        fmt.Println("Erreur lors du parsing du formulaire:", err)
        http.Error(w, "Erreur lors du parsing du formulaire", http.StatusBadRequest)
        return
    }
    
    file, handler, err := r.FormFile("profile_image")
    if err != nil {
        fmt.Println("Erreur lors de la récupération du fichier:", err)
        http.Redirect(w, r, "/profile?msg=file_upload_error", http.StatusFound)
        return
    }
    defer file.Close()
    
    buff := make([]byte, 512)
    _, err = file.Read(buff)
    if err != nil {
        fmt.Println("Erreur lors de la lecture du fichier:", err)
        http.Redirect(w, r, "/profile?msg=file_read_error", http.StatusFound)
        return
    }
    
    filetype := http.DetectContentType(buff)
    if !strings.HasPrefix(filetype, "image/") {
        fmt.Println("Type de fichier non autorisé:", filetype)
        http.Redirect(w, r, "/profile?msg=file_type_error", http.StatusFound)
        return
    }
    
    file.Seek(0, io.SeekStart)
    
    filename := fmt.Sprintf("%d_%s", time.Now().Unix(), handler.Filename)
    
    uploadDir := "public/uploads/profiles"
    err = os.MkdirAll(uploadDir, 0755)
    if err != nil {
        fmt.Println("Erreur lors de la création du dossier:", err)
        http.Redirect(w, r, "/profile?msg=directory_error", http.StatusFound)
        return
    }
    
    dst, err := os.Create(filepath.Join(uploadDir, filename))
    if err != nil {
        fmt.Println("Erreur lors de la création du fichier:", err)
        http.Redirect(w, r, "/profile?msg=file_create_error", http.StatusFound)
        return
    }
    defer dst.Close()
    
    if _, err = io.Copy(dst, file); err != nil {
        fmt.Println("Erreur lors de la copie du fichier:", err)
        http.Redirect(w, r, "/profile?msg=file_copy_error", http.StatusFound)
        return
    }
    
    cookie, _ := r.Cookie("SESSION")
    username := databaseAPI.GetUser(database, cookie.Value)
    
    success := databaseAPI.UpdateProfileImage(database, username, filename)
    if !success {
        fmt.Println("Erreur lors de la mise à jour de l'image de profil dans la DB")
        http.Redirect(w, r, "/profile?msg=db_update_error", http.StatusFound)
        return
    }
    
    http.Redirect(w, r, "/profile?msg=profile_image_updated", http.StatusFound)
}