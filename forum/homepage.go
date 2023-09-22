package forum

import (
	"net/http"
	"text/template"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func executePosts() ([]Post, error) {
	var posts []Post //local struct - don't change as it duplicates the posts for some reason.
	rows, err := DB.Query("SELECT id, title, content, created_at FROM posts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.Time)
		if err != nil {
			return nil, err
		}
		// Format the datetime string
		t, err := time.Parse("2006-01-02T15:04:05.999999999-07:00", post.Time)
		if err != nil {
			return nil, err
		}
		post.Time = t.Format("January 2, 2006, 15:04:05")
		// make post URLs
		post.URL = "/post/" + post.ID
		posts = append(posts, post)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	// reverse posts
	posts = reverse(posts)
	return posts, nil
}

// reverse posts (latest first)
func reverse(s []Post) []Post {
	//runes := []rune(s)
	length := len(s)
	for i, j := 0, length-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

// serve homepage
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user is already logged in
	existingSessionID, _ := r.Cookie("session")
	isLoggedIn := existingSessionID != nil

	posts, err := executePosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := HomePageData{
		Posts:      posts,
		IsLoggedIn: isLoggedIn, // Pass the IsLoggedIn information to the template
	}

	tmpl, err := template.ParseFiles("home.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handle filtered posts
func FilteredPostsHandler(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	// Retrieve the posts based on the selected category
	filteredPosts, err := getPostsByCategory(category)
	if err != nil {
		http.Error(w, "Could not fetch posts", http.StatusInternalServerError)
		return
	}

	var data struct {
		Category      string
		FilteredPosts []Post // Use a slice of Post
	}

	data.Category = category
	data.FilteredPosts = filteredPosts

	tmpl, err := template.ParseFiles("filteredPosts.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Render the template with the filtered posts data
	err = tmpl.ExecuteTemplate(w, "filteredPosts.html", data)
	if err != nil {
		http.Error(w, "Internal Server Error - homepage", http.StatusInternalServerError)
		return
	}
}

// retrieve posts by their category
func getPostsByCategory(category string) ([]Post, error) {
	rows, err := DB.Query("SELECT id, title, content, created_at FROM posts WHERE category_id = ?", category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post

	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.Time)
		if err != nil {
			return nil, err
		}

		// Format the datetime string
		t, err := time.Parse("2006-01-02T15:04:05.999999999-07:00", post.Time)
		if err != nil {
			return nil, err
		}
		post.Time = t.Format("January 2, 2006, 15:04:05")

		// make post URLs
		post.URL = "/post/" + post.ID

		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}
