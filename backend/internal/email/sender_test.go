package email

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDevelopmentOutboxWritesPrivateMIMEMessage(t *testing.T) {
	root := t.TempDir()
	sender := OutboxSender{Root: root}
	err := sender.Send(context.Background(), Message{To: "player@example.com", Subject: "Verify", Template: "email_verification", Link: "https://app.example/verify?token=secret"})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("outbox entries=%d err=%v", len(entries), err)
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("message permissions=%o, want 600", info.Mode().Perm())
	}
	data, err := os.ReadFile(filepath.Join(root, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	message := string(data)
	for _, required := range []string{"To: player@example.com", "Subject: Verify", "Content-Type: text/html", "https://app.example/verify?token=secret"} {
		if !strings.Contains(message, required) {
			t.Fatalf("MIME message missing %q", required)
		}
	}
}
