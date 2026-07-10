package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	URL             string
	MigrationsDir   string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type Database struct {
	DB *sql.DB
}

func Open(ctx context.Context, cfg Config) (*Database, error) {
	if cfg.URL == "" || !strings.HasPrefix(strings.ToLower(cfg.URL), "postgres") {
		return nil, errors.New("postgres url is required")
	}
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, err
	}
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Database{DB: db}, nil
}

func (d *Database) ValidateMigrations(ctx context.Context, migrationsDir string) error {
	if d == nil || d.DB == nil {
		return errors.New("postgres database is not open")
	}
	if migrationsDir == "" {
		migrationsDir = "./migrations"
	}
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no migrations found in %s", migrationsDir)
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if !strings.Contains(strings.ToLower(string(content)), "create table") {
			return fmt.Errorf("migration %s does not define schema", filepath.Base(file))
		}
	}
	var one int
	return d.DB.QueryRowContext(ctx, "SELECT 1").Scan(&one)
}

func (d *Database) Close() error {
	if d == nil || d.DB == nil {
		return nil
	}
	return d.DB.Close()
}
