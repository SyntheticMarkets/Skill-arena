package storage

import (
	"context"
	"os"
	"testing"
)

func TestS3CompatibleIntegration(t *testing.T) {
	endpoint := os.Getenv("SKILL_ARENA_TEST_S3_ENDPOINT")
	bucket := os.Getenv("SKILL_ARENA_TEST_S3_BUCKET")
	accessKey := os.Getenv("SKILL_ARENA_TEST_S3_ACCESS_KEY")
	secretKey := os.Getenv("SKILL_ARENA_TEST_S3_SECRET_KEY")
	if endpoint == "" || bucket == "" || accessKey == "" || secretKey == "" {
		t.Skip("S3-compatible integration environment is not configured")
	}
	store := S3CompatibleStore{
		Endpoint: endpoint, Bucket: bucket, AccessKey: accessKey, SecretKey: secretKey,
		Region: "us-east-1",
	}
	ctx := context.Background()
	if err := store.Health(ctx); err != nil {
		t.Fatal(err)
	}
	key := "integration/sprint-3-object-storage.txt"
	want := []byte("Skill Arena Sprint 3 object storage integrity")
	if err := store.Put(ctx, key, want, "text/plain"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Delete(context.Background(), key); err != nil {
			t.Errorf("cleanup object: %v", err)
		}
	})
	got, err := store.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("object content=%q, want %q", got, want)
	}
}
