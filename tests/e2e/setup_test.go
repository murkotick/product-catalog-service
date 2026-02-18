package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	databasepb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instancepb "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"

	"github.com/murkotick/product-catalog-service/internal/app/product/queries"
	"github.com/murkotick/product-catalog-service/internal/app/product/repo"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/activate_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/apply_discount"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/create_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/update_product"
	"github.com/murkotick/product-catalog-service/internal/pkg/clock"
	committer "github.com/murkotick/product-catalog-service/internal/pkg/committer"
)

var (
	spClient *spanner.Client
	clk      *clock.FakeClock

	createUC   *create_product.Interactor
	updateUC   *update_product.Interactor
	activateUC *activate_product.Interactor
	applyDisUC *apply_discount.Interactor

	readModel *queries.SpannerReadModel

	dbName string
)

func TestMain(m *testing.M) {
	// Keep time in UTC everywhere.
	now := time.Now().UTC().Truncate(time.Second)
	clk = clock.NewFake(now)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	// Default emulator host if not provided.
	if os.Getenv("SPANNER_EMULATOR_HOST") == "" {
		_ = os.Setenv("SPANNER_EMULATOR_HOST", "localhost:9010")
	}

	projectID := env("SPANNER_PROJECT_ID", "test-project")
	instanceID := env("SPANNER_INSTANCE_ID", "emulator-instance")
	// Use a unique database per "go test" run to avoid flakiness and id collisions.
	databaseID := fmt.Sprintf("e2e_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))

	parent := fmt.Sprintf("projects/%s", projectID)
	instName := fmt.Sprintf("%s/instances/%s", parent, instanceID)
	dbName = fmt.Sprintf("%s/databases/%s", instName, databaseID)

	instAdmin, err := instance.NewInstanceAdminClient(ctx)
	if err != nil {
		panic(fmt.Sprintf("instance admin client: %v", err))
	}
	defer instAdmin.Close()

	dbAdmin, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		panic(fmt.Sprintf("database admin client: %v", err))
	}
	defer dbAdmin.Close()

	// Ensure instance exists.
	ensureInstance(ctx, instAdmin, parent, instName, instanceID)

	// Create database.
	createStmt := fmt.Sprintf("CREATE DATABASE `%s`", databaseID)
	op, err := dbAdmin.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          instName,
		CreateStatement: createStmt,
	})
	if err != nil {
		// If the DB already exists (unlikely with UUID) just continue.
		if status.Code(err) != codes.AlreadyExists {
			panic(fmt.Sprintf("CreateDatabase: %v", err))
		}
	} else {
		if _, err := op.Wait(ctx); err != nil {
			panic(fmt.Sprintf("CreateDatabase wait: %v", err))
		}
	}

	// Apply DDL.
	ddlPath := filepath.Join("..", "..", "migrations", "001_initial_schema.sql")
	ddl, err := os.ReadFile(ddlPath)
	if err != nil {
		panic(fmt.Sprintf("read %s: %v", ddlPath, err))
	}
	stmts := splitDDL(string(ddl))
	ddlOp, err := dbAdmin.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   dbName,
		Statements: stmts,
	})
	if err != nil {
		panic(fmt.Sprintf("UpdateDatabaseDdl: %v", err))
	}
	if err := ddlOp.Wait(ctx); err != nil {
		panic(fmt.Sprintf("UpdateDatabaseDdl wait: %v", err))
	}

	// Data client.
	spClient, err = spanner.NewClient(ctx, dbName)
	if err != nil {
		panic(fmt.Sprintf("spanner.NewClient: %v", err))
	}

	// Wire dependencies.
	prodRepo := repo.NewProductRepo()
	outboxRepo := repo.NewOutboxRepo()
	cm := committer.NewAdapter(spClient)
	readModel = queries.NewSpannerReadModel(spClient)

	createUC = create_product.NewInteractor(prodRepo, outboxRepo, cm, clk)
	updateUC = update_product.NewInteractor(prodRepo, outboxRepo, cm, readModel, clk)
	activateUC = activate_product.NewInteractor(prodRepo, outboxRepo, cm, readModel, clk)
	applyDisUC = apply_discount.NewInteractor(prodRepo, outboxRepo, cm, readModel, clk)

	code := m.Run()

	spClient.Close()

	// Best-effort cleanup (emulator only).
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel2()
	_ = deleteDatabase(ctx2, dbAdmin, dbName)

	os.Exit(code)
}

func ensureInstance(ctx context.Context, admin *instance.InstanceAdminClient, parent, instName, instanceID string) {
	_, err := admin.GetInstance(ctx, &instancepb.GetInstanceRequest{Name: instName})
	if err == nil {
		return
	}
	if status.Code(err) != codes.NotFound {
		panic(fmt.Sprintf("GetInstance: %v", err))
	}

	// Create instance for emulator.
	op, err := admin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     parent,
		InstanceId: instanceID,
		Instance: &instancepb.Instance{
			Config:      fmt.Sprintf("%s/instanceConfigs/emulator-config", parent),
			DisplayName: "E2E Test Instance",
			NodeCount:   1,
		},
	})
	if err != nil {
		if status.Code(err) != codes.AlreadyExists {
			panic(fmt.Sprintf("CreateInstance: %v", err))
		}
		return
	}
	if _, err := op.Wait(ctx); err != nil {
		panic(fmt.Sprintf("CreateInstance wait: %v", err))
	}
}

func deleteDatabase(ctx context.Context, admin *database.DatabaseAdminClient, db string) error {
	err := admin.DropDatabase(ctx, &databasepb.DropDatabaseRequest{Database: db})
	return err
}

func splitDDL(sql string) []string {
	// normalize line endings
	sql = strings.ReplaceAll(sql, "\r\n", "\n")
	parts := strings.Split(sql, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		stmt := strings.TrimSpace(p)
		if stmt == "" {
			continue
		}
		out = append(out, stmt)
	}
	return out
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func requireEmulator(t *testing.T) {
	// A quick sanity check so failures are easier to understand.
	require.NotEmpty(t, os.Getenv("SPANNER_EMULATOR_HOST"), "SPANNER_EMULATOR_HOST must be set (e.g. localhost:9010)")
}
