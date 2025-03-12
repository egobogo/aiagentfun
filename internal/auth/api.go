```go
package auth

import (
	"net/http"
	"encoding/json"
	"errors"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var users = map[string]string{}

func Register(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil || user.Username == "" || user.Password == "" {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if _, exists := users[user.Username]; exists {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}
	users[user.Username] = user.Password
	w.WriteHeader(http.StatusCreated)
}

func Login(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil || user.Username == "" || user.Password == "" {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	storedPassword, exists := users[user.Username]
	if !exists || storedPassword != user.Password {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}
```

!!internal/auth/api_test.go!!
```go
package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegister_Success(t *testing.T) {
	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	Register(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, res.StatusCode)
	}
}

func TestRegister_UserExists(t *testing.T) {
	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	Register(w, req) // First registration

	req = httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(reqBody)) // Second registration
	w = httptest.NewRecorder()

	Register(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, res.StatusCode)
	}
}

func TestRegister_InvalidInput(t *testing.T) {
	reqBody := `{"username":"","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	Register(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}
}

func TestLogin_Success(t *testing.T) {
	reqBody := `{"username":"testuser","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	Register(w, req) // Register user first

	reqBody = `{"username":"testuser","password":"password123"}`
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(reqBody))
	w = httptest.NewRecorder()

	Login(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, res.StatusCode)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	reqBody := `{"username":"testuser","password":"wrongpassword"}`
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	Login(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, res.StatusCode)
	}
}

func TestLogin_InvalidInput(t *testing.T) {
	reqBody := `{"username":"","password":""}`
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	Login(w, req)

	res := w.Result()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, res.StatusCode)
	}
}
```