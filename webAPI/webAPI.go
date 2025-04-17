package webAPI

import (
	"FORUM-GO/databaseAPI"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"
	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	IsLoggedIn bool
	Username   string
}

type HomePage struct {
	User              User
	Categories        []string
	Icons             []string
	PostsByCategories [][]databaseAPI.Post
	MFAEnabled        bool  // Champ pour l'état de l'A2F
}

type PostsPage struct {
	User  User
	Title string
	Posts []databaseAPI.Post
	Icon  string
}

type PostPage struct {
	User User
	Post databaseAPI.Post
}

type EditPostPageData struct {
	User User
	Post databaseAPI.Post
}

var database *sql.DB

func SetDatabase(db *sql.DB) {
	database = db
}


func Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if checkUserLoggedIn(r) {
		cookie, _ := r.Cookie("SESSION")
		username := databaseAPI.GetUser(database, cookie.Value)
		
		
		mfaEnabled, _ := databaseAPI.IsMFAEnabled(database, username)
		
		payload := HomePage{
			User:              User{IsLoggedIn: true, Username: username},
			Categories:        databaseAPI.GetCategories(database),
			Icons:             databaseAPI.GetCategoriesIcons(database),
			PostsByCategories: databaseAPI.GetPostsByCategories(database),
			MFAEnabled:        mfaEnabled,  // Indique si MFA est activé
		}
		t, _ := template.ParseGlob("public/HTML/*.html")
		t.ExecuteTemplate(w, "forum.html", payload)
		return
	}
	payload := HomePage{
		User:              User{IsLoggedIn: false},
		Categories:        databaseAPI.GetCategories(database),
		Icons:             databaseAPI.GetCategoriesIcons(database),
		PostsByCategories: databaseAPI.GetPostsByCategories(database),
		MFAEnabled:        false,  // Par défaut, MFA n'est pas activé
	}
	t, _ := template.ParseGlob("public/HTML/*.html")
	t.ExecuteTemplate(w, "forum.html", payload)
	return
}


func DisplayPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	post := databaseAPI.GetPost(database, id)
	comments := databaseAPI.GetComments(database, id)
	
	
	var isUserLoggedIn bool
	var username string
	if checkUserLoggedIn(r) {
		cookie, _ := r.Cookie("SESSION")
		username = databaseAPI.GetUser(database, cookie.Value)
		isUserLoggedIn = true
		
		
		for i := range comments {
			
			hasLiked := false
			rows, err := database.Query(`
				SELECT COUNT(*) FROM comment_likes 
				JOIN users ON comment_likes.user_id = users.id
				WHERE comment_likes.comment_id = ? AND users.username = ?
			`, comments[i].Id, username)
			
			if err == nil {
				if rows.Next() {
					var count int
					rows.Scan(&count)
					hasLiked = count > 0
				}
				rows.Close()
			}
			
			comments[i].UserLiked = hasLiked
			
			
			hasDisliked := false
			rows, err = database.Query(`
				SELECT COUNT(*) FROM comment_dislikes 
				JOIN users ON comment_dislikes.user_id = users.id
				WHERE comment_dislikes.comment_id = ? AND users.username = ?
			`, comments[i].Id, username)
			
			if err == nil {
				if rows.Next() {
					var count int
					rows.Scan(&count)
					hasDisliked = count > 0
				}
				rows.Close()
			}
			
			comments[i].UserDisliked = hasDisliked
			
			
			rows, err = database.Query("SELECT COUNT(*) FROM comment_likes WHERE comment_id = ?", comments[i].Id)
			if err == nil {
				if rows.Next() {
					var count int
					rows.Scan(&count)
					comments[i].Likes = count
				}
				rows.Close()
			}
			
			
			rows, err = database.Query("SELECT COUNT(*) FROM comment_dislikes WHERE comment_id = ?", comments[i].Id)
			if err == nil {
				if rows.Next() {
					var count int
					rows.Scan(&count)
					comments[i].Dislikes = count
				}
				rows.Close()
			}
		}
	} else {
		isUserLoggedIn = false
		
		
		for i := range comments {
			
			rows, err := database.Query("SELECT COUNT(*) FROM comment_likes WHERE comment_id = ?", comments[i].Id)
			if err == nil {
				if rows.Next() {
					var count int
					rows.Scan(&count)
					comments[i].Likes = count
				}
				rows.Close()
			}
			
			
			rows, err = database.Query("SELECT COUNT(*) FROM comment_dislikes WHERE comment_id = ?", comments[i].Id)
			if err == nil {
				if rows.Next() {
					var count int
					rows.Scan(&count)
					comments[i].Dislikes = count
				}
				rows.Close()
			}
		}
	}
	
	post.Comments = comments
	
	
	post.Images = databaseAPI.GetPostImages(database, post.Id)
	
	payload := PostPage{
		Post: post,
		User: User{IsLoggedIn: isUserLoggedIn, Username: username},
	}
	
	t, _ := template.ParseGlob("public/HTML/*.html")
	t.ExecuteTemplate(w, "detail.html", payload)
}


func GetPostsByApi(w http.ResponseWriter, r *http.Request) {
	method := r.URL.Query().Get("by")
	
	
	var username string
	var isLoggedIn bool
	
	
	if checkUserLoggedIn(r) {
		cookie, _ := r.Cookie("SESSION")
		username = databaseAPI.GetUser(database, cookie.Value)
		isLoggedIn = true
	}
	
	
	if method == "category" {
		category := r.URL.Query().Get("category")
		posts := databaseAPI.GetPostsByCategory(database, category)
		payload := PostsPage{
			Title: "Publications dans la catégorie " + category,
			Posts: posts,
			Icon:  databaseAPI.GetCategoryIcon(database, category),
		}
		if isLoggedIn {
			payload.User = User{IsLoggedIn: true, Username: username}
		}
		t, _ := template.ParseGlob("public/HTML/*.html")
		t.ExecuteTemplate(w, "posts.html", payload)
		return
	}
	
	
	if method == "myposts" {
		if isLoggedIn {
			posts := databaseAPI.GetPostsByUser(database, username)
			payload := PostsPage{
				User:  User{IsLoggedIn: true, Username: username},
				Title: "Mes publications",
				Posts: posts,
				Icon:  "fa-user",
			}
			t, _ := template.ParseGlob("public/HTML/*.html")
			t.ExecuteTemplate(w, "posts.html", payload)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	
	
	if method == "liked" {
		if isLoggedIn {
			posts := databaseAPI.GetLikedPosts(database, username)
			payload := PostsPage{
				User:  User{IsLoggedIn: true, Username: username},
				Title: "Publications aimées",
				Posts: posts,
				Icon:  "fa-heart",
			}
			t, _ := template.ParseGlob("public/HTML/*.html")
			t.ExecuteTemplate(w, "posts.html", payload)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	
	
	if method == "advanced" {
		if !isLoggedIn {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		
		
		keyword := r.URL.Query().Get("keyword")
		category := r.URL.Query().Get("category")
		sortBy := r.URL.Query().Get("sort")
		onlyMine := r.URL.Query().Get("mine") == "true"
		onlyLiked := r.URL.Query().Get("liked") == "true"
		
		
		posts := databaseAPI.GetAdvancedFilteredPosts(
			database, 
			category, 
			keyword, 
			sortBy, 
			username, 
			onlyMine, 
			onlyLiked,
		)
		
		payload := PostsPage{
			User:  User{IsLoggedIn: true, Username: username},
			Title: "Résultats de recherche",
			Posts: posts,
			Icon:  "fa-search",
		}
		
		t, _ := template.ParseGlob("public/HTML/*.html")
		t.ExecuteTemplate(w, "posts.html", payload)
		return
	}
	
	
	http.Error(w, "Méthode de filtrage non reconnue", http.StatusBadRequest)
}

func NewPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !checkUserLoggedIn(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	t, _ := template.ParseGlob("public/HTML/*.html")
	t.ExecuteTemplate(w, "createThread.html", nil)
}


func EditPostPage(w http.ResponseWriter, r *http.Request) {
    if !checkUserLoggedIn(r) {
        http.Redirect(w, r, "/login", http.StatusFound)
        return
    }
    
    id := r.URL.Query().Get("postId")
    post := databaseAPI.GetPost(database, id)
    
    
    cookie, _ := r.Cookie("SESSION")
    username := databaseAPI.GetUser(database, cookie.Value)
    
    postId, err := strconv.Atoi(id)
    if err != nil {
        http.Error(w, "ID de post invalide", http.StatusBadRequest)
        return
    }
    
    if !databaseAPI.IsPostOwner(database, username, postId) {
        http.Error(w, "Non autorisé - Vous n'êtes pas le propriétaire de ce post", http.StatusUnauthorized)
        return
    }
    
    payload := EditPostPageData{
        Post: post,
        User: User{IsLoggedIn: true, Username: username},
    }
    
    t, err := template.ParseFiles("public/HTML/editpost.html")
    if err != nil {
        fmt.Println("Erreur lors du chargement du template:", err)
        http.Error(w, "Erreur interne du serveur", http.StatusInternalServerError)
        return
    }
    
    err = t.Execute(w, payload)
    if err != nil {
        fmt.Println("Erreur lors de l'exécution du template:", err)
        http.Error(w, "Erreur interne du serveur", http.StatusInternalServerError)
    }
}


func CommentLikeApi(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
        return
    }

    
    if !checkUserLoggedIn(r) {
        http.Redirect(w, r, "/login", http.StatusFound)
        return
    }

    
    cookie, err := r.Cookie("SESSION")
    if err != nil {
        http.Error(w, "Erreur de session", http.StatusUnauthorized)
        return
    }

    username := databaseAPI.GetUser(database, cookie.Value)
    
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Erreur lors du parsing du formulaire", http.StatusBadRequest)
        return
    }
    
    commentIdStr := r.FormValue("commentId")
    postIdStr := r.FormValue("postId")
    action := r.FormValue("action") 
    
    commentId, err := strconv.Atoi(commentIdStr)
    if err != nil {
        http.Error(w, "ID de commentaire invalide", http.StatusBadRequest)
        return
    }

   
    var userId int
    err = database.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userId)
    if err != nil {
        http.Error(w, "Erreur de récupération d'utilisateur: "+err.Error(), http.StatusInternalServerError)
        return
    }
    
    if action == "like" {
        
        var likeExists bool
        err = database.QueryRow("SELECT COUNT(*) > 0 FROM comment_likes WHERE comment_id = ? AND user_id = ?", 
            commentId, userId).Scan(&likeExists)
        if err != nil {
            http.Error(w, "Erreur lors de la vérification du like: "+err.Error(), http.StatusInternalServerError)
            return
        }
        
        
        var dislikeExists bool
        err = database.QueryRow("SELECT COUNT(*) > 0 FROM comment_dislikes WHERE comment_id = ? AND user_id = ?", 
            commentId, userId).Scan(&dislikeExists)
        if err != nil {
            http.Error(w, "Erreur lors de la vérification du dislike: "+err.Error(), http.StatusInternalServerError)
            return
        }
        
       
        if dislikeExists {
            _, err = database.Exec("DELETE FROM comment_dislikes WHERE comment_id = ? AND user_id = ?", 
                commentId, userId)
            if err != nil {
                http.Error(w, "Erreur lors de la suppression du dislike: "+err.Error(), http.StatusInternalServerError)
                return
            }
        }
        
        if likeExists {
            
            _, err = database.Exec("DELETE FROM comment_likes WHERE comment_id = ? AND user_id = ?", 
                commentId, userId)
        } else {
            
            _, err = database.Exec("INSERT INTO comment_likes (comment_id, user_id, created_at) VALUES (?, ?, ?)", 
                commentId, userId, time.Now().Format("2006-01-02 15:04:05"))
        }
    } else if action == "dislike" {
        
        var dislikeExists bool
        err = database.QueryRow("SELECT COUNT(*) > 0 FROM comment_dislikes WHERE comment_id = ? AND user_id = ?", 
            commentId, userId).Scan(&dislikeExists)
        if err != nil {
            http.Error(w, "Erreur lors de la vérification du dislike: "+err.Error(), http.StatusInternalServerError)
            return
        }
        
        
        var likeExists bool
        err = database.QueryRow("SELECT COUNT(*) > 0 FROM comment_likes WHERE comment_id = ? AND user_id = ?", 
            commentId, userId).Scan(&likeExists)
        if err != nil {
            http.Error(w, "Erreur lors de la vérification du like: "+err.Error(), http.StatusInternalServerError)
            return
        }
        
        
        if likeExists {
            _, err = database.Exec("DELETE FROM comment_likes WHERE comment_id = ? AND user_id = ?", 
                commentId, userId)
            if err != nil {
                http.Error(w, "Erreur lors de la suppression du like: "+err.Error(), http.StatusInternalServerError)
                return
            }
        }
        
        if dislikeExists {
            
            _, err = database.Exec("DELETE FROM comment_dislikes WHERE comment_id = ? AND user_id = ?", 
                commentId, userId)
        } else {
            
            _, err = database.Exec("INSERT INTO comment_dislikes (comment_id, user_id, created_at) VALUES (?, ?, ?)", 
                commentId, userId, time.Now().Format("2006-01-02 15:04:05"))
        }
    }
    
    if err != nil {
        http.Error(w, "Erreur lors du traitement de la réaction: "+err.Error(), http.StatusInternalServerError)
        return
    }

    
    http.Redirect(w, r, "/post?id="+postIdStr, http.StatusSeeOther)
}


func checkUserLoggedIn(r *http.Request) bool {
    cookie, err := r.Cookie("SESSION")
    if err != nil {
        return false
    }
    cookieExists := databaseAPI.CheckCookie(database, cookie.Value)
    if !cookieExists {
        return false
    }
    expires := databaseAPI.GetExpires(database, cookie.Value)
    
    expiresTime, err := time.Parse("2006-01-02 15:04:05", expires)
    if err != nil {
        return false
    }
    
    return !time.Now().After(expiresTime)
}


func inArray(input string, array []string) bool {
	for _, v := range array {
		if v == input {
			return true
		}
	}
	return false
}