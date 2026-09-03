package services

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/verbeux-ai/whatsmiau/env"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.uber.org/zap"
)

var sqlStoreInstance *sqlstore.Container

func prepareSQLiteURL(rawURL string) string {
	cleanURL := rawURL

	// Convert file:/absolute/path to standard SQLite file:///absolute/path
	if strings.HasPrefix(cleanURL, "file:/") && !strings.HasPrefix(cleanURL, "file:///") {
		cleanURL = "file://" + strings.TrimPrefix(cleanURL, "file:")
	}

	// Extract filesystem path to guarantee parent directory exists
	filePath := cleanURL
	if strings.HasPrefix(filePath, "file://") {
		if u, err := url.Parse(filePath); err == nil && u.Path != "" {
			filePath = u.Path
		} else {
			filePath = strings.TrimPrefix(filePath, "file://")
		}
	} else if strings.HasPrefix(filePath, "file:") {
		filePath = strings.TrimPrefix(filePath, "file:")
	}

	if idx := strings.Index(filePath, "?"); idx != -1 {
		filePath = filePath[:idx]
	}

	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0777); err != nil {
			zap.L().Warn("failed to ensure sqlite database directory", zap.String("dir", dir), zap.Error(err))
		}
	}

	return cleanURL
}

func SQLStore() *sqlstore.Container {
	ctx, c := context.WithTimeout(context.Background(), 10*time.Second)
	defer c()

	if sqlStoreInstance == nil {
		dbURL := env.Env.DBURL
		if env.Env.DBDialect == "sqlite3" {
			dbURL = prepareSQLiteURL(dbURL)
		}

		container, err := sqlstore.New(ctx, env.Env.DBDialect, dbURL, nil)
		if err != nil {
			zap.L().Panic("failed to start sqlstore", zap.Error(err))
		}

		sqlStoreInstance = container
	}

	return sqlStoreInstance
}

