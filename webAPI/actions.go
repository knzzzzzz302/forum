package webAPI

import (
	"FORUM-GO/databaseAPI"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Vote struct {
	PostId int
	Vote   int
}


func CreatePostApi(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	
	if err := r.ParseMultipartForm(10 << 20); err != nil { 
		http.Error(w, fmt.Sprintf("Erreur de ParseMultipartForm(): %v", err), http.StatusBadRequest)
		return
	}

	
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	
	cookie, err := r.Cookie("SESSION")
	if err != nil {
		http.Error(w, "Erreur de cookie SESSION", http.StatusUnauthorized)
		return
	}

	username := databaseAPI.GetUser(database, cookie.Value)
	title := r.FormValue("title")
	content := r.FormValue("content")
	categories := r.Form["categories[]"]

	
	validCategories := databaseAPI.GetCategories(database)
	for _, category := range categories {
		if !inArray(category, validCategories) {
			http.Error(w, fmt.Sprintf("Catégorie invalide : %s", category), http.StatusBadRequest)
			return
		}
	}

	
	stringCategories := strings.Join(categories, ",")

	
	now := time.Now()
	postId := databaseAPI.CreatePost(database, username, title, stringCategories, content, now)
	
	
	files := r.MultipartForm.File["images"]
	for _, fileHeader := range files {
		
		file, err := fileHeader.Open()
		if err != nil {
			fmt.Printf("Erreur lors de l'ouverture du fichier: %v\n", err)
			continue
		}
		defer file.Close()
		
		
		buff := make([]byte, 512)
		_, err = file.Read(buff)
		if err != nil {
			fmt.Printf("Erreur lors de la lecture du fichier: %v\n", err)
			continue
		}
		
		filetype := http.DetectContentType(buff)
		if !strings.HasPrefix(filetype, "image/") {
			fmt.Printf("Type de fichier non autorisé: %s\n", filetype)
			continue
		}
		
		
		file.Seek(0, io.SeekStart)
		
		
		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), fileHeader.Filename)
		filepath := filepath.Join("public/uploads/posts", filename)
		
		
		dst, err := os.Create(filepath)
		if err != nil {
			fmt.Printf("Erreur lors de la création du fichier: %v\n", err)
			continue
		}
		defer dst.Close()
		
		
		if _, err = io.Copy(dst, file); err != nil {
			fmt.Printf("Erreur lors de la copie du fichier: %v\n", err)
			continue
		}
		
		
		imagePath := "/public/uploads/posts/" + filename
		err = databaseAPI.AddPostImage(database, int(postId), imagePath)
		if err != nil {
			fmt.Printf("Erreur lors de l'enregistrement de l'image dans la DB: %v\n", err)
		}
	}
	
	fmt.Printf("Post créé par %s avec le titre %s à %s\n", username, title, now.Format("2006-01-02 15:04:05"))

	
	http.Redirect(w, r, "/filter?by=myposts", http.StatusFound)
	return
}


func CommentsApi(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("Erreur de ParseForm(): %v", err), http.StatusBadRequest)
		return
	}

	
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	
	cookie, err := r.Cookie("SESSION")
	if err != nil {
		http.Error(w, "Erreur de cookie SESSION", http.StatusUnauthorized)
		return
	}

	username := databaseAPI.GetUser(database, cookie.Value)
	postId := r.FormValue("postId")
	content := r.FormValue("content")
	now := time.Now()

	
	postIdInt, err := strconv.Atoi(postId)
	if err != nil {
		http.Error(w, "Post ID invalide", http.StatusBadRequest)
		return
	}

	
	databaseAPI.AddComment(database, username, postIdInt, content, now)
	fmt.Printf("Commentaire créé par %s sur le post %s à %s\n", username, postId, now.Format("2006-01-02 15:04:05"))

	
	http.Redirect(w, r, "/post?id="+postId, http.StatusFound)
	return
}


func VoteApi(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Error(w, "Authentification requise", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Erreur de traitement du formulaire", http.StatusBadRequest)
		return
	}

	cookie, _ := r.Cookie("SESSION")
	username := databaseAPI.GetUser(database, cookie.Value)
	postId := r.FormValue("postId")
	postIdInt, err := strconv.Atoi(postId)
	if err != nil {
		http.Error(w, "Identifiant de post invalide", http.StatusBadRequest)
		return
	}

	vote := r.FormValue("vote")
	voteInt, err := strconv.Atoi(vote)
	if err != nil || (voteInt != 1 && voteInt != -1) {
		http.Error(w, "Valeur de vote invalide", http.StatusBadRequest)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	logPrefix := fmt.Sprintf("Vote - Utilisateur %s, Post %s, %s: ", username, postId, now)

	if voteInt == 1 {
		if databaseAPI.HasUpvoted(database, username, postIdInt) {
			
			databaseAPI.RemoveVote(database, postIdInt, username)
			databaseAPI.DecreaseUpvotes(database, postIdInt)
			fmt.Println(logPrefix + "Suppression d'un vote positif")
		} else if databaseAPI.HasDownvoted(database, username, postIdInt) {
			
			databaseAPI.RemoveVote(database, postIdInt, username)
			databaseAPI.DecreaseDownvotes(database, postIdInt)
			databaseAPI.AddVote(database, postIdInt, username, 1)
			databaseAPI.IncreaseUpvotes(database, postIdInt)
			fmt.Println(logPrefix + "Changement de vote négatif à positif")
		} else {
			
			databaseAPI.AddVote(database, postIdInt, username, 1)
			databaseAPI.IncreaseUpvotes(database, postIdInt)
			fmt.Println(logPrefix + "Nouveau vote positif")
		}
	} else { 
		if databaseAPI.HasDownvoted(database, username, postIdInt) {
			
			databaseAPI.RemoveVote(database, postIdInt, username)
			databaseAPI.DecreaseDownvotes(database, postIdInt)
			fmt.Println(logPrefix + "Suppression d'un vote négatif")
		} else if databaseAPI.HasUpvoted(database, username, postIdInt) {
			
			databaseAPI.RemoveVote(database, postIdInt, username)
			databaseAPI.DecreaseUpvotes(database, postIdInt)
			databaseAPI.AddVote(database, postIdInt, username, -1)
			databaseAPI.IncreaseDownvotes(database, postIdInt)
			fmt.Println(logPrefix + "Changement de vote positif à négatif")

			} else {
					
					databaseAPI.AddVote(database, postIdInt, username, -1)
					databaseAPI.IncreaseDownvotes(database, postIdInt)
					fmt.Println(logPrefix + "Nouveau vote négatif")
				}
			}
		
			http.Redirect(w, r, "/post?id="+strconv.Itoa(postIdInt), http.StatusFound)
		}
		
		
		func EditPostHandler(w http.ResponseWriter, r *http.Request) {
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
		
			
			cookie, err := r.Cookie("SESSION")
			if err != nil {
				http.Error(w, "Erreur de cookie SESSION", http.StatusUnauthorized)
				return
			}
		
			
			username := databaseAPI.GetUser(database, cookie.Value)
			postIdStr := r.FormValue("postId")
			title := r.FormValue("title")
			content := r.FormValue("content")
			categories := r.Form["categories[]"]
		
			
			postId, err := strconv.Atoi(postIdStr)
			if err != nil {
				http.Error(w, "ID de post invalide", http.StatusBadRequest)
				return
			}
		
			
			if !databaseAPI.IsPostOwner(database, username, postId) {
				http.Error(w, "Non autorisé - Vous n'êtes pas le propriétaire de ce post", http.StatusUnauthorized)
				return
			}
		
			
			var stringCategories string
			if len(categories) == 0 {
				post := databaseAPI.GetPost(database, postIdStr)
				stringCategories = strings.Join(post.Categories, ",")
			} else {
				stringCategories = strings.Join(categories, ",")
			}
		
			
			success := databaseAPI.EditPost(database, postId, title, stringCategories, content)
			if !success {
				http.Error(w, "Erreur lors de la mise à jour du post", http.StatusInternalServerError)
				return
			}
		
			
			http.Redirect(w, r, "/post?id="+postIdStr, http.StatusFound)
		}
		
		
		func DeletePostHandler(w http.ResponseWriter, r *http.Request) {
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
		
			
			cookie, err := r.Cookie("SESSION")
			if err != nil {
				http.Error(w, "Erreur de cookie SESSION", http.StatusUnauthorized)
				return
			}
		
			
			username := databaseAPI.GetUser(database, cookie.Value)
			postIdStr := r.FormValue("postId")
		
			
			postId, err := strconv.Atoi(postIdStr)
			if err != nil {
				http.Error(w, "ID de post invalide", http.StatusBadRequest)
				return
			}
		
			
			if !databaseAPI.IsPostOwner(database, username, postId) {
				http.Error(w, "Non autorisé - Vous n'êtes pas le propriétaire de ce post", http.StatusUnauthorized)
				return
			}
		
			
			success := databaseAPI.DeletePost(database, postId)
			if !success {
				http.Error(w, "Erreur lors de la suppression du post", http.StatusInternalServerError)
				return
			}
		
			
			http.Redirect(w, r, "/", http.StatusFound)
		}
		
		
		func EditCommentHandler(w http.ResponseWriter, r *http.Request) {
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
		
			
			cookie, err := r.Cookie("SESSION")
			if err != nil {
				http.Error(w, "Erreur de cookie SESSION", http.StatusUnauthorized)
				return
			}
		
			
			username := databaseAPI.GetUser(database, cookie.Value)
			commentIdStr := r.FormValue("commentId")
			postIdStr := r.FormValue("postId")
			content := r.FormValue("content")
		
			
			commentId, err := strconv.Atoi(commentIdStr)
			if err != nil {
				http.Error(w, "ID de commentaire invalide", http.StatusBadRequest)
				return
			}
		
			
			if !databaseAPI.IsCommentOwner(database, username, commentId) {
				http.Error(w, "Non autorisé - Vous n'êtes pas le propriétaire de ce commentaire", http.StatusUnauthorized)
				return
			}
		
			
			success := databaseAPI.EditComment(database, commentId, content)
			if !success {
				http.Error(w, "Erreur lors de la mise à jour du commentaire", http.StatusInternalServerError)
				return
			}
		
			
			http.Redirect(w, r, "/post?id="+postIdStr, http.StatusFound)
		}
		
		
		func DeleteCommentHandler(w http.ResponseWriter, r *http.Request) {
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
		
			
			cookie, err := r.Cookie("SESSION")
			if err != nil {
				http.Error(w, "Erreur de cookie SESSION", http.StatusUnauthorized)
				return
			}
		
			
			username := databaseAPI.GetUser(database, cookie.Value)
			commentIdStr := r.FormValue("commentId")
			postIdStr := r.FormValue("postId")
		
			
			commentId, err := strconv.Atoi(commentIdStr)
			if err != nil {
				http.Error(w, "ID de commentaire invalide", http.StatusBadRequest)
				return
			}
		
			
			if !databaseAPI.IsCommentOwner(database, username, commentId) {
				http.Error(w, "Non autorisé - Vous n'êtes pas le propriétaire de ce commentaire", http.StatusUnauthorized)
				return
			}
		
			
			success := databaseAPI.DeleteComment(database, commentId)
			if !success {
				http.Error(w, "Erreur lors de la suppression du commentaire", http.StatusInternalServerError)
				return
			}
		
			
			http.Redirect(w, r, "/post?id="+postIdStr, http.StatusFound)
		}