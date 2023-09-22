package forum

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// check cookies to see if user is logged in properly
func checkCookies(w http.ResponseWriter, r *http.Request) {
	sessionCookie, err := r.Cookie("session")
	if err != nil {
		// If there is an error, it means the session cookie was not found
		// Redirect user to login page
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if sessionCookie.Value == "" {
		// If the session cookie is empty, the user is not logged in
		// Redirect user to login page
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
}

// CREATE POSTS FUNCTION
func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	// Check session cookie
	checkCookies(w, r)

	if r.Method == http.MethodGet {
		// Serve create post page
		http.ServeFile(w, r, "createPost.html")
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Could not parse form", http.StatusBadRequest)
		return
	}

	_, userId, err := GetCookieValue(r)
	if err != nil {
		http.Error(w, "cookie not found", http.StatusBadRequest)
		return
	}

	titleContent := r.Form.Get("postTitle")
	postContent := r.Form.Get("postContent")

	if titleContent == "" || postContent == "" {
		fmt.Fprintln(w, "Error - please ensure title and post content fields are not empty!")
		return
	}

	// Get selected categories
	categories := r.Form["postCategories"]

	// Convert categories to a comma-separated string
	categoriesString := strings.Join(categories, ",")
	fmt.Printf("Post categories: %s\n", categoriesString)

	//added
	dateCreated := time.Now()
	fmt.Println(userId)
	userIdInt, err := strconv.Atoi(userId)
	if err != nil {
		http.Error(w, "Could not convert", http.StatusInternalServerError)
		return
	}
	// - added placeholders and userintid
	_, err = DB.Exec("INSERT INTO posts (user_id, title, content, category_id, created_at) VALUES (?, ?, ?, ?, ?)", userIdInt, titleContent, postContent, categoriesString, dateCreated)
	if err != nil {
		log.Println(err)
		http.Error(w, "Could not create post", http.StatusInternalServerError)
		return
	}

	fmt.Println("Post successfully created!")

	// Redirect the user to the homepage
	http.Redirect(w, r, "/", http.StatusFound)
}

func GetCookieValue(r *http.Request) (string, string, error) {
	//- indices represent the split to cookie and value
	cookie, err := r.Cookie("session")
	if err != nil {
		return "", "", err
	}
	value := strings.Split(cookie.Value, "&")

	return value[0], value[1], nil
}

// get post ID
func getPostByID(postID string) (*Post, error) {
	//added
	// Adjusted the SELECT query to also get the `dislike_count`
	row := DB.QueryRow("SELECT id, title, content, created_at, likes_count, dislikes_count FROM posts WHERE id = ?", postID)
	var post Post
	// Added &post.DislikeCount at the end
	err := row.Scan(&post.ID, &post.Title, &post.Content, &post.Time, &post.LikesCount, &post.DislikeCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("post not found")
		}
		return nil, err
	}
	// row := DB.QueryRow("SELECT id, title, content, created_at, likes_count FROM posts WHERE id = ?", postID)
	// var post Post
	// err := row.Scan(&post.ID, &post.Title, &post.Content, &post.Time, &post.LikesCount)
	// if err != nil {
	// 	if err == sql.ErrNoRows {
	// 		return nil, errors.New("post not found")
	// 	}
	// 	return nil, err
	// }
	// Format the datetime string
	t, err := time.Parse("2006-01-02T15:04:05.999999999-07:00", post.Time)
	if err != nil {
		return nil, err
	}
	post.Time = t.Format("January 2, 2006, 15:04:05")
	// make post URLs
	post.URL = "/post/" + post.ID
	return &post, nil
}

func getCommentsByPostID(postID string) ([]Comment, error) {
	comments := []Comment{} // creating an empty slice to store comments from the database //i've also added postID and userID to the comment struct
	rows, err := DB.Query("SELECT user_id, post_id, content, created_at FROM comments WHERE post_id = ?", postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.UserID, &comment.PostID, &comment.Content, &comment.Time)
		if err != nil {
			return nil, err
		}
		t, err := time.Parse("2006-01-02T15:04:05.999999999-07:00", comment.Time)
		if err != nil {
			return nil, err
		}
		comment.Time = t.Format("January 2, 2006, 15:04:05")
		comments = append(comments, comment)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

func PostPageHandler(w http.ResponseWriter, r *http.Request) {
	// Get post id from the URL path
	postIDStr := strings.TrimPrefix(r.URL.Path, "/post/")

	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}
	var comments []Comment

	// Get the post data by calling the getPostByID function or fetching it from the database
	post, err := getPostByID(postIDStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Get the likes count from the post variable
	likesCount := post.LikesCount
	dislikeCount := post.DislikeCount

	//get comments by postID -
	comments, err = getCommentsByPostID(postIDStr)
	if err != nil {
		http.Error(w, "Could not fetch comments", http.StatusInternalServerError)
		return
	}

	// Assuming your Post struct has a field named PostID
	var data struct {
		PostID   int
		Post     Post
		Comments []Comment
		Likes    int
		Dislikes int
		Success  bool // Add the Success field to indicate if the comment was successfully posted
	}

	data.PostID = postID
	data.Post = *post // Use the dereferenced post pointer
	data.Comments = comments
	data.Success = r.URL.Query().Get("success") == "1"
	data.Likes = likesCount
	data.Dislikes = dislikeCount

	fmt.Println(likesCount, "likes count")
	fmt.Println(dislikeCount, "dislike count")

	tmpl, err := template.ParseFiles("postPage.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Render the template with the data
	// ! error flag
	err = tmpl.ExecuteTemplate(w, "postPage.html", data)
	if err != nil {
		http.Error(w, "Internal Server Error - posts", http.StatusInternalServerError)
		return
	}
}
