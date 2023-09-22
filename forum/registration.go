package forum

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	// "github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// generate random session ID
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func init() {
	rand.NewSource(time.Now().UnixNano())
}

func generateUserID(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "register.html")
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Could not parse form", http.StatusBadRequest)
		return
	}
	email := r.Form.Get("email")
	username := r.Form.Get("username")
	password := r.Form.Get("password")

	if email == "" || username == "" || password == "" {
		http.Error(w, "Please fill out all fields", http.StatusBadRequest)
		return
	}

	// Generate a new session ID for the user
	sessionID := generateUserID(10)

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err)
		http.Error(w, "Could not hash password", http.StatusInternalServerError)
		return
	}

	// Insert the user into the database with the new session ID
	_, err = DB.Exec("INSERT INTO Users (Email, Username, Password, SessionID) VALUES (?, ?, ?, ?)", email, username, hashedPassword, sessionID)
	if err != nil {
		log.Println(err)
		http.Error(w, "Could not create user", http.StatusInternalServerError)
		return
	}
	fmt.Println("User registered")

	// Set a session cookie to indicate that the user is logged in
	http.SetCookie(w, &http.Cookie{
		Name:    "session",
		Value:   sessionID,
		Expires: time.Now().Add(1 * time.Hour),
		Path:    "/",
	})

	// Redirect the user to the homepage, where the logout button will be displayed
	http.Redirect(w, r, "/", http.StatusFound)
}

// func removeOldSession(sessionCookieValue string) {
// 	// Split the sessionCookieValue to get the session ID
// 	parts := strings.Split(sessionCookieValue, "&")
// 	if len(parts) != 2 {
// 		// Invalid session cookie format
// 		return
// 	}

// 	sessionID := parts[0]

// 	// Implement code to remove or invalidate the old session in your database
// 	query := "DELETE FROM Users WHERE SessionID = ?"
// 	_, err := DB.Exec(query, sessionID)
// 	if err != nil {
// 		// Handle the error, e.g., log it
// 		log.Println("Error removing old session:", err)
// 	}
// }

// handle login + session cookies
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the user already has an active session
	existingSessionID, _ := r.Cookie("session")
	if existingSessionID != nil {
		LogoutHandler(w, r)
	}

	// Generate a new session ID
	newSessionID := generateUserID(10)
	fmt.Println(newSessionID)

	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "login.html")
		return
	}

	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "login.html")
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Could not parse form", http.StatusBadRequest)
		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")
	if email == "" || password == "" {
		http.Error(w, "Please fill out all fields", http.StatusBadRequest)
		return
	}

	var userId int
	var storedPassword []byte // holds the hashed password from the database
	err = DB.QueryRow("SELECT ID, password FROM Users WHERE email = ?", email).Scan(&userId, &storedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "No user found with this email", http.StatusUnauthorized)
		} else {
			log.Println(err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	err = bcrypt.CompareHashAndPassword(storedPassword, []byte(password))
	if err != nil {
		http.Error(w, "Incorrect password", http.StatusUnauthorized)
		return
	}

	// check if sessionID already exists
	// query := "SELECT COUNT(*) FROM Users WHERE Email = ? AND sessionID = ?"

	// var count int
	// err = DB.QueryRow(query, email, newSessionID).Scan(&count)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// // if sessionID already exists, user stays logged in
	// if count > 0 {
	// 	fmt.Printf("SessionID '%s' exists for Email '%s'\n", newSessionID, email)
	// } else { // if sessionID does not exist, user is logged out
	// 	fmt.Printf("SessionID '%s' does not exist for Email '%s'\n", newSessionID, email)
	// }

	// insert session ID into database
	_, err = DB.Exec("UPDATE Users SET SessionID = ? WHERE Email = ?", newSessionID, email)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// // Set browser cookies to store login
	// http.SetCookie(w, &http.Cookie{
	// 	Name:  "user",
	// 	Value: email,
	// })

	// Store the new session ID in a cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session",
		Value:   newSessionID + "&" + strconv.Itoa(userId),
		Expires: time.Now().Add(1 * time.Hour),
		Path:    "/",
	})

	// Redirect the user to the homepage
	http.Redirect(w, r, "/", http.StatusFound)

	// // Create a new cookie with the session ID
	// cookie := http.Cookie{
	// 	Name:    "session",
	// 	Value:   newSessionID + "&" + strconv.Itoa(userId),
	// 	Expires: time.Now().Add(1 * time.Hour), // Extend cookie expiration time
	// 	Path:    "/",
	// }

	// Set the cookie in the response headers
	// http.SetCookie(w, &cookie)

	// Redirect the user to the homepage
	// http.Redirect(w, r, "/", http.StatusFound)
}

// handle logging out
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the session data from the database
	sessionCookie, err := r.Cookie("session")
	if err == nil {
		sessionID := strings.Split(sessionCookie.Value, "&")[0]
		clearSessionFromDB(sessionID)
	}

	// Clear session and user cookies
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		MaxAge: -1, // Expire immediately
		Path:   "/",
	})
	http.SetCookie(w, &http.Cookie{
		Name:   "user",
		Value:  "",
		MaxAge: -1, // Expire immediately
		Path:   "/",
	})

	// Redirect the user to the login page
	http.Redirect(w, r, "/login", http.StatusFound)
}

// clearSessionFromDB removes the session data from the database
func clearSessionFromDB(sessionID string) {
	// Implement code to clear the session from the database by setting SessionID to NULL
	query := "UPDATE Users SET SessionID = NULL WHERE SessionID = ?"
	_, err := DB.Exec(query, sessionID)
	if err != nil {
		// Handle the error, e.g., log it
		log.Println("Error clearing session from the database:", err)
	}
}
