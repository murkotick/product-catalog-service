package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	databasepb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
)

// A tiny migration helper that applies the DDL in migrations/001_initial_schema.sql
// to a Cloud Spanner database (typically the emulator for local dev).
//
// Usage (emulator):
//
//	set SPANNER_EMULATOR_HOST=localhost:9010
//	set SPANNER_DATABASE=projects/test-project/instances/emulator-instance/databases/test-db
//	go run ./cmd/migrate
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db := os.Getenv("SPANNER_DATABASE")
	if db == "" {
		log.Fatal("SPANNER_DATABASE is required (e.g. projects/test-project/instances/emulator-instance/databases/test-db)")
	}

	ddlPath := filepath.Join("migrations", "001_initial_schema.sql")
	stmts, err := readDDLStatements(ddlPath)
	if err != nil {
		log.Fatalf("read DDL: %v", err)
	}
	if len(stmts) == 0 {
		log.Fatalf("no DDL statements found in %s", ddlPath)
	}

	admin, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		log.Fatalf("database admin client: %v", err)
	}
	defer admin.Close()

	op, err := admin.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   db,
		Statements: stmts,
	})
	if err != nil {
		log.Fatalf("UpdateDatabaseDdl: %v", err)
	}

	if err := op.Wait(ctx); err != nil {
		log.Fatalf("UpdateDatabaseDdl wait: %v", err)
	}

	fmt.Printf("Applied %d DDL statements to %s\n", len(stmts), db)
}

func readDDLStatements(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Normalize line endings for Windows-authored files.
	sql := strings.ReplaceAll(string(b), "\r\n", "\n")

	parts := strings.Split(sql, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		stmt := strings.TrimSpace(p)
		if stmt == "" {
			continue
		}
		out = append(out, stmt)
	}
	return out, nil
}
