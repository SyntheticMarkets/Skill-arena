package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestS3CompatibleStoreLifecycleAndSigning(t *testing.T) {
	var mu sync.Mutex
	objects := map[string][]byte{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 Credential=test-access/") {
			http.Error(w, "signature required", http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/financial" && r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/financial/") {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodPut:
			objects[r.URL.Path], _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			data, exists := objects[r.URL.Path]
			if !exists {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(data)
		case http.MethodDelete:
			delete(objects, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	store := S3CompatibleStore{
		Endpoint: server.URL, Bucket: "financial", AccessKey: strings.Join([]string{"test", "access"}, "-"),
		SecretKey: strings.Repeat("s", 32), Region: "af-south-1",
	}
	ctx := context.Background()
	if err := store.Health(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, "statements/user/one.csv", []byte("minor\n100\n"), "text/csv"); err != nil {
		t.Fatal(err)
	}
	data, err := store.Get(ctx, "statements/user/one.csv")
	if err != nil || string(data) != "minor\n100\n" {
		t.Fatalf("data=%q err=%v", data, err)
	}
	if err := store.Delete(ctx, "statements/user/one.csv"); err != nil {
		t.Fatal(err)
	}
}

func TestLocalStoreRejectsEscapingObjectKeys(t *testing.T) {
	store := LocalStore{Root: t.TempDir()}
	if err := store.Put(context.Background(), "../outside", []byte("no"), "text/plain"); err == nil {
		t.Fatal("expected path traversal to fail")
	}
}
