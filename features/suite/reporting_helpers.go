//go:build bdd

package suite

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"google.golang.org/grpc/metadata"
)

// reportingDBTables перечисляет таблицы reporting-service, которые трункаются между сценариями.
var reportingDBTables = []string{
	"sla_reports",
}

// reportingJWTSecret — общий с reporting-service секрет для BDD-окружения.
const reportingJWTSecret = alertJWTSecret

// createReportingDB создаёт отдельную БД для reporting-service в общем postgres контейнере.
func createReportingDB(ctx context.Context, pg *tcpostgres.PostgresContainer) (string, func(), error) {
	baseDSN, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return "", nil, fmt.Errorf("get base connection string: %w", err)
	}

	dbName := fmt.Sprintf("reporting_bdd_%d", randSuffix())

	adminDB, err := sql.Open("postgres", baseDSN)
	if err != nil {
		return "", nil, fmt.Errorf("open admin db: %w", err)
	}
	defer func() {
		//nolint:errcheck // закрываем admin-соединение
		_ = adminDB.Close()
	}()

	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName)); err != nil {
		return "", nil, fmt.Errorf("create database %q: %w", dbName, err)
	}

	u, err := url.Parse(baseDSN)
	if err != nil {
		return "", nil, fmt.Errorf("parse base dsn: %w", err)
	}
	u.Path = "/" + dbName
	reportingDSN := u.String()

	cleanup := func() {
		cleanupCtx := context.Background()
		db, err := sql.Open("postgres", baseDSN)
		if err != nil {
			return
		}
		defer func() {
			//nolint:errcheck // закрываем admin-соединение
			_ = db.Close()
		}()
		//nolint:errcheck // best-effort drop
		_, _ = db.ExecContext(cleanupCtx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
	}

	return reportingDSN, cleanup, nil
}

// reportingServiceDir возвращает рабочую директорию reporting-service.
func reportingServiceDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "backend", "reporting-service")
}

// reportingAuthTokenFor возвращает HS256 JWT токен для reporting-service.
// Reporting-service ожидает claims user_id (uuid) и стандартные exp/iat.
func reportingAuthTokenFor(userID uuid.UUID) string {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	now := time.Now().Unix()
	claims := map[string]any{
		"user_id": userID.String(),
		"email":   "bdd@test.example.com",
		"tier":    "Free",
		"iat":     now,
		"exp":     now + 3600,
	}
	hb, _ := json.Marshal(header)
	cb, _ := json.Marshal(claims)
	enc := base64.RawURLEncoding
	signing := enc.EncodeToString(hb) + "." + enc.EncodeToString(cb)
	mac := hmac.New(sha256.New, []byte(reportingJWTSecret))
	mac.Write([]byte(signing))
	sig := enc.EncodeToString(mac.Sum(nil))
	return signing + "." + sig
}

// reportingAuthCtx добавляет Authorization metadata для вызовов reporting-service.
func reportingAuthCtx(ctx context.Context, userID uuid.UUID) context.Context {
	if userID == uuid.Nil {
		return ctx
	}
	md := metadata.Pairs(
		"authorization", "Bearer "+reportingAuthTokenFor(userID),
	)
	return metadata.NewOutgoingContext(ctx, md)
}
