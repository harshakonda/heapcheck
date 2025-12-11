// Package server demonstrates escape analysis in HTTP handlers.
// Common patterns in web applications that cause heap allocations.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

// User represents a user in the system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Response is a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// =============================================================================
// Pattern: Interface Boxing in Handlers
// =============================================================================

// HandleUserBad - interface{} in Response causes boxing
func HandleUserBad(w http.ResponseWriter, r *http.Request) {
	user := User{ID: 1, Name: "John", Email: "john@example.com"}

	// Response.Data is interface{} - causes user to escape
	resp := Response{
		Success: true,
		Data:    user, // ESCAPES - boxed to interface{}
	}

	json.NewEncoder(w).Encode(resp)
}

// UserResponse is a typed response - no interface boxing
type UserResponse struct {
	Success bool `json:"success"`
	Data    User `json:"data,omitempty"`
}

// HandleUserGood - typed response avoids interface boxing
func HandleUserGood(w http.ResponseWriter, r *http.Request) {
	user := User{ID: 1, Name: "John", Email: "john@example.com"}

	resp := UserResponse{
		Success: true,
		Data:    user, // No interface - may still escape via json but cleaner
	}

	json.NewEncoder(w).Encode(resp)
}

// =============================================================================
// Pattern: String Formatting in Handlers
// =============================================================================

// HandleErrorBad uses fmt.Sprintf - causes interface boxing
func HandleErrorBad(w http.ResponseWriter, r *http.Request, code int) {
	msg := fmt.Sprintf("Error code: %d", code) // code boxed to interface{}
	http.Error(w, msg, http.StatusBadRequest)
}

// HandleErrorGood uses strconv - no boxing
func HandleErrorGood(w http.ResponseWriter, r *http.Request, code int) {
	msg := "Error code: " + strconv.Itoa(code) // no interface
	http.Error(w, msg, http.StatusBadRequest)
}

// =============================================================================
// Pattern: Middleware Closure Captures
// =============================================================================

// LoggingMiddlewareBad captures logger in closure
func LoggingMiddlewareBad(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// logger captured by closure - may cause issues
			logger.Log(r.URL.Path) // captured variable
			next.ServeHTTP(w, r)
		})
	}
}

// Logger is a simple logger
type Logger struct {
	prefix string
}

// Log logs a message
func (l *Logger) Log(msg string) {
	fmt.Println(l.prefix + msg)
}

// LoggingMiddlewareGood uses struct with logger field
type loggingMiddleware struct {
	logger *Logger
	next   http.Handler
}

func (m *loggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.logger.Log(r.URL.Path)
	m.next.ServeHTTP(w, r)
}

// NewLoggingMiddleware creates middleware without closure capture
func NewLoggingMiddleware(logger *Logger, next http.Handler) http.Handler {
	return &loggingMiddleware{logger: logger, next: next}
}

// =============================================================================
// Pattern: Request Body Handling
// =============================================================================

// CreateUserBad allocates new User for each request
func CreateUserBad(w http.ResponseWriter, r *http.Request) {
	var user User // allocated per request
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// process user...
}

// For high-throughput: use sync.Pool
var userPool = &sync.Pool{
	New: func() interface{} {
		return new(User)
	},
}

// CreateUserPooled reuses User objects
func CreateUserPooled(w http.ResponseWriter, r *http.Request) {
	user := userPool.Get().(*User)
	defer userPool.Put(user)

	// Reset user fields
	*user = User{}

	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// process user...
}
