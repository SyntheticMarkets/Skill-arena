package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"skill-arena/internal/config"
	"skill-arena/internal/db"
	"skill-arena/internal/handlers"
	"skill-arena/internal/models"

	"golang.org/x/crypto/bcrypt"
)

func benchmarkRequest(handler http.Handler, method, path string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	payload, _ := json.Marshal(body)
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://app.skillarena.test")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func BenchmarkSprint1Registration(b *testing.B) {
	store, err := db.New(context.Background(), b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	cfg := authTestConfig("")
	handler := handlers.RegisterHandler(store, cfg)
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		response := benchmarkRequest(handler, http.MethodPost, "/api/v1/auth/register", map[string]any{"email": fmt.Sprintf("benchmark-registration-%d@example.com", index), "password": "StrongPassword!42", "country": "ZA", "dateOfBirth": "1990-01-01", "acceptTerms": true, "acceptFairPlay": true})
		if response.Code != http.StatusCreated {
			b.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
	}
}

func BenchmarkSprint1Login(b *testing.B) {
	store, err := db.New(context.Background(), b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	password := "StrongPassword!42"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := models.NewUser("benchmark-login-user", "benchmark-login@example.com", string(hash))
	user.EmailVerified = true
	if err := store.CreateUser(context.Background(), user); err != nil {
		b.Fatal(err)
	}
	cfg := authTestConfig("")
	handler := handlers.LoginHandler(store, cfg)
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		response := benchmarkRequest(handler, http.MethodPost, "/api/v1/auth/login", map[string]string{"email": user.Email, "password": password})
		if response.Code != http.StatusOK {
			b.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
	}
}

func BenchmarkSprint1SessionAuthentication(b *testing.B) {
	store, err := db.New(context.Background(), b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	password := "StrongPassword!42"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := models.NewUser("benchmark-session-user", "benchmark-session@example.com", string(hash))
	user.EmailVerified = true
	if err := store.CreateUser(context.Background(), user); err != nil {
		b.Fatal(err)
	}
	cfg := authTestConfig("")
	login := benchmarkRequest(handlers.LoginHandler(store, cfg), http.MethodPost, "/api/v1/auth/login", map[string]string{"email": user.Email, "password": password})
	var access *http.Cookie
	for _, cookie := range login.Result().Cookies() {
		if cookie.Name == cfg.Settings.Security.AccessCookieName {
			access = cookie
			break
		}
	}
	if access == nil {
		b.Fatal("login did not return an access cookie")
	}
	handler := handlers.AuthMiddleware(store, cfg, handlers.SessionStatusHandler(store))
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		response := benchmarkRequest(handler, http.MethodGet, "/api/v1/auth/session", nil, access)
		if response.Code != http.StatusOK {
			b.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
	}
}

func BenchmarkSprint1PostgresIdentityQueries(b *testing.B) {
	databaseURL := os.Getenv("SKILL_ARENA_TEST_POSTGRES_URL")
	if databaseURL == "" {
		b.Skip("SKILL_ARENA_TEST_POSTGRES_URL is not configured")
	}
	store, err := db.NewWithOptions(context.Background(), db.Options{DatabaseURL: databaseURL, Environment: "development", Storage: config.StorageSettings{LocalRoot: b.TempDir()}})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = store.Close(context.Background()) })
	email := fmt.Sprintf("benchmark-postgres-%d@example.com", time.Now().UnixNano())
	hash, _ := bcrypt.GenerateFromPassword([]byte("StrongPassword!42"), bcrypt.MinCost)
	user := models.NewUser("", email, string(hash))
	user.EmailVerified = true
	if err := store.CreateUser(context.Background(), user); err != nil {
		b.Fatal(err)
	}
	refresh := db.NewRefreshToken()
	session, err := store.CreateAuthSession(context.Background(), user.ID, refresh, "benchmark", "127.0.0.1", time.Hour)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("GetUserByEmail", func(b *testing.B) {
		for index := 0; index < b.N; index++ {
			if _, err := store.GetUserByEmail(context.Background(), email); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ValidateSession", func(b *testing.B) {
		for index := 0; index < b.N; index++ {
			if _, _, err := store.ValidateAuthSession(context.Background(), session.ID, user.ID); err != nil {
				b.Fatal(err)
			}
		}
	})
}
