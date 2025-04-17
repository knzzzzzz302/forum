package databaseAPI

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
	"strings"
	"time"
)

func GetPost(database *sql.DB, id string) Post {
	rows, _ := database.Query("SELECT username, title, categories, content, created_at, upvotes, downvotes FROM posts WHERE id = ?", id)
	var post Post
	post.Id, _ = strconv.Atoi(id)
	for rows.Next() {
		catString := ""
		rows.Scan(&post.Username, &post.Title, &catString, &post.Content, &post.CreatedAt, &post.UpVotes, &post.DownVotes)
		categoriesArray := strings.Split(catString, ",")
		post.Categories = categoriesArray
	}
	
	post.Images = GetPostImages(database, post.Id)
	
	post.ProfileImage = GetProfileImage(database, post.Username)
	
	return post
}

func GetPostImages(database *sql.DB, postId int) []string {
	rows, err := database.Query("SELECT image_path FROM post_images WHERE post_id = ?", postId)
	if err != nil {
		return []string{}
	}
	defer rows.Close()
	
	var images []string
	for rows.Next() {
		var imagePath string
		err := rows.Scan(&imagePath)
		if err != nil {
			continue
		}
		images = append(images, imagePath)
	}
	
	return images
}

func AddPostImage(database *sql.DB, postId int, imagePath string) error {
	statement, err := database.Prepare("INSERT INTO post_images (post_id, image_path) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer statement.Close()
	
	_, err = statement.Exec(postId, imagePath)
	return err
}

func GetComments(database *sql.DB, id string) []Comment {
	rows, _ := database.Query("SELECT id, username, content, created_at FROM comments WHERE post_id = ?", id)
	var comments []Comment
	for rows.Next() {
		var comment Comment
		rows.Scan(&comment.Id, &comment.Username, &comment.Content, &comment.CreatedAt)
		comment.ProfileImage = GetProfileImage(database, comment.Username)
		comments = append(comments, comment)
	}
	return comments
}

func GetPostsByCategory(database *sql.DB, category string) []Post {
	rows, _ := database.Query("SELECT id, username, title, categories, content, created_at, upvotes, downvotes FROM posts WHERE categories LIKE ?", "%"+category+"%")
	var posts []Post
	for rows.Next() {
		var post Post
		var catString string
		rows.Scan(&post.Id, &post.Username, &post.Title, &catString, &post.Content, &post.CreatedAt, &post.UpVotes, &post.DownVotes)
		post.Categories = strings.Split(catString, ",")
		post.Images = GetPostImages(database, post.Id)
		posts = append(posts, post)
	}
	return posts
}

func GetPostsByCategories(database *sql.DB) [][]Post {
	categories := GetCategories(database)
	var posts [][]Post
	for _, category := range categories {
		posts = append(posts, GetPostsByCategory(database, category))
	}
	return posts
}

func GetPostsByUser(database *sql.DB, username string) []Post {
	rows, _ := database.Query("SELECT id, username, title, categories, content, created_at, upvotes, downvotes FROM posts WHERE username = ?", username)
	var posts []Post
	for rows.Next() {
		var post Post
		var catString string
		rows.Scan(&post.Id, &post.Username, &post.Title, &catString, &post.Content, &post.CreatedAt, &post.UpVotes, &post.DownVotes)
		post.Categories = strings.Split(catString, ",")
		post.Images = GetPostImages(database, post.Id)
		posts = append(posts, post)
	}
	return posts
}

func GetLikedPosts(database *sql.DB, username string) []Post {
	rows, _ := database.Query("SELECT id, username, title, categories, content, created_at, upvotes, downvotes FROM posts WHERE id IN (SELECT post_id FROM votes WHERE username = ? AND vote = 1)", username)
	var posts []Post
	for rows.Next() {
		var post Post
		var catString string
		rows.Scan(&post.Id, &post.Username, &post.Title, &catString, &post.Content, &post.CreatedAt, &post.UpVotes, &post.DownVotes)
		post.Categories = strings.Split(catString, ",")
		post.Images = GetPostImages(database, post.Id)
		posts = append(posts, post)
	}
	return posts
}

func GetCategories(database *sql.DB) []string {
	rows, _ := database.Query("SELECT name FROM categories")
	var categories []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		categories = append(categories, name)
	}
	return categories
}

func GetCategoriesIcons(database *sql.DB) []string {
	rows, _ := database.Query("SELECT icon FROM categories")
	var icons []string
	for rows.Next() {
		var icon string
		rows.Scan(&icon)
		icons = append(icons, icon)
	}
	return icons
}

func GetCategoryIcon(database *sql.DB, category string) string {
	rows, _ := database.Query("SELECT icon FROM categories WHERE name = ?", category)
	var icon string
	for rows.Next() {
		rows.Scan(&icon)
	}
	return icon
}

func CreatePost(database *sql.DB, username string, title string, categories string, content string, createdAt time.Time) int64 {
	createdAtString := createdAt.Format("2006-01-02 15:04:05")
	statement, _ := database.Prepare("INSERT INTO posts (username, title, categories, content, created_at, upvotes, downvotes) VALUES (?, ?, ?, ?, ?, ?, ?)")
	result, _ := statement.Exec(username, title, categories, content, createdAtString, 0, 0)
	postId, _ := result.LastInsertId()
	return postId
}

func AddComment(database *sql.DB, username string, postId int, content string, createdAt time.Time) {
	createdAtString := createdAt.Format("2006-01-02 15:04:05")
	statement, _ := database.Prepare("INSERT INTO comments (username, post_id, content, created_at) VALUES (?, ?, ?, ?)")
	statement.Exec(username, postId, content, createdAtString)
}

func EditPost(database *sql.DB, postId int, title string, categories string, content string) bool {
	statement, err := database.Prepare("UPDATE posts SET title = ?, categories = ?, content = ? WHERE id = ?")
	if err != nil {
		return false
	}
	_, err = statement.Exec(title, categories, content, postId)
	if err != nil {
		return false
	}
	return true
}

func DeletePost(database *sql.DB, postId int) bool {
	statementImages, err := database.Prepare("DELETE FROM post_images WHERE post_id = ?")
	if err != nil {
		return false
	}
	_, err = statementImages.Exec(postId)
	if err != nil {
		return false
	}
	
	statementVotes, err := database.Prepare("DELETE FROM votes WHERE post_id = ?")
	if err != nil {
		return false
	}
	_, err = statementVotes.Exec(postId)
	if err != nil {
		return false
	}
	
	statementComments, err := database.Prepare("DELETE FROM comments WHERE post_id = ?")
	if err != nil {
		return false
	}
	_, err = statementComments.Exec(postId)
	if err != nil {
		return false
	}
	
	statementPost, err := database.Prepare("DELETE FROM posts WHERE id = ?")
	if err != nil {
		return false
	}
	_, err = statementPost.Exec(postId)
	if err != nil {
		return false
	}
	
	return true
}

func EditComment(database *sql.DB, commentId int, content string) bool {
	statement, err := database.Prepare("UPDATE comments SET content = ? WHERE id = ?")
	if err != nil {
		return false
	}
	_, err = statement.Exec(content, commentId)
	if err != nil {
		return false
	}
	return true
}

func DeleteComment(database *sql.DB, commentId int) bool {
	statement, err := database.Prepare("DELETE FROM comments WHERE id = ?")
	if err != nil {
		return false
	}
	_, err = statement.Exec(commentId)
	if err != nil {
		return false
	}
	return true
}

func IsPostOwner(database *sql.DB, username string, postId int) bool {
	var count int
	err := database.QueryRow("SELECT COUNT(*) FROM posts WHERE id = ? AND username = ?", postId, username).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

func IsCommentOwner(database *sql.DB, username string, commentId int) bool {
	var count int
	err := database.QueryRow("SELECT COUNT(*) FROM comments WHERE id = ? AND username = ?", commentId, username).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

func GetPostsByDate(database *sql.DB, ascending bool) []Post {
	orderDirection := "DESC"
	if ascending {
		orderDirection = "ASC"
	}
	
	query := fmt.Sprintf("SELECT id, username, title, categories, content, created_at, upvotes, downvotes FROM posts ORDER BY created_at %s", orderDirection)
	rows, _ := database.Query(query)
	var posts []Post
	for rows.Next() {
		var post Post
		var catString string
		rows.Scan(&post.Id, &post.Username, &post.Title, &catString, &post.Content, &post.CreatedAt, &post.UpVotes, &post.DownVotes)
		post.Categories = strings.Split(catString, ",")
		post.Images = GetPostImages(database, post.Id)
		posts = append(posts, post)
	}
	return posts
}

func GetPostsByPopularity(database *sql.DB) []Post {
	rows, _ := database.Query("SELECT id, username, title, categories, content, created_at, upvotes, downvotes FROM posts ORDER BY (upvotes - downvotes) DESC")
	var posts []Post
	for rows.Next() {
		var post Post
		var catString string
		rows.Scan(&post.Id, &post.Username, &post.Title, &catString, &post.Content, &post.CreatedAt, &post.UpVotes, &post.DownVotes)
		post.Categories = strings.Split(catString, ",")
		post.Images = GetPostImages(database, post.Id)
		posts = append(posts, post)
	}
	return posts
}

func GetPostsByKeyword(database *sql.DB, keyword string) []Post {
	rows, _ := database.Query("SELECT id, username, title, categories, content, created_at, upvotes, downvotes FROM posts WHERE title LIKE ? OR content LIKE ?", 
		"%"+keyword+"%", "%"+keyword+"%")
	var posts []Post
	for rows.Next() {
		var post Post
		var catString string
		rows.Scan(&post.Id, &post.Username, &post.Title, &catString, &post.Content, &post.CreatedAt, &post.UpVotes, &post.DownVotes)
		post.Categories = strings.Split(catString, ",")
		post.Images = GetPostImages(database, post.Id)
		posts = append(posts, post)
	}
	return posts
}

func GetAdvancedFilteredPosts(database *sql.DB, category string, keyword string, sortBy string, username string, onlyMine bool, onlyLiked bool) []Post {
	query := "SELECT id, username, title, categories, content, created_at, upvotes, downvotes FROM posts WHERE 1=1"
	var args []interface{}
	
	if category != "" {
		query += " AND categories LIKE ?"
		args = append(args, "%"+category+"%")
	}
	
	if keyword != "" {
		query += " AND (title LIKE ? OR content LIKE ?)"
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}
	
	if onlyMine {
		query += " AND username = ?"
		args = append(args, username)
	}
	
	if onlyLiked {
		query += " AND id IN (SELECT post_id FROM votes WHERE username = ? AND vote = 1)"
		args = append(args, username)
	}
	
	switch sortBy {
	case "date_asc":
		query += " ORDER BY created_at ASC"
	case "date_desc":
		query += " ORDER BY created_at DESC"
	case "popularity":
		query += " ORDER BY (upvotes - downvotes) DESC"
	}
	
	rows, _ := database.Query(query, args...)
	var posts []Post
	for rows.Next() {
		var post Post
		var catString string
		rows.Scan(&post.Id, &post.Username, &post.Title, &catString, &post.Content, &post.CreatedAt, &post.UpVotes, &post.DownVotes)
		post.Categories = strings.Split(catString, ",")
		post.Images = GetPostImages(database, post.Id)
		posts = append(posts, post)
	}
	return posts
}
