package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc"

	"github.com/murkotick/product-catalog-service/internal/app/product/queries"
	"github.com/murkotick/product-catalog-service/internal/app/product/queries/get_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/queries/list_products"
	"github.com/murkotick/product-catalog-service/internal/app/product/repo"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/activate_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/apply_discount"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/create_product"
	"github.com/murkotick/product-catalog-service/internal/app/product/usecases/update_product"
	"github.com/murkotick/product-catalog-service/internal/pkg/clock"
	committer "github.com/murkotick/product-catalog-service/internal/pkg/committer"
	grpcproduct "github.com/murkotick/product-catalog-service/internal/transport/grpc/product"
	productv1 "github.com/murkotick/product-catalog-service/proto/product/v1"
)

func main() {
	addr := env("GRPC_ADDR", ":50051")
	spannerDB := env("SPANNER_DATABASE", "projects/test-project/instances/emulator-instance/databases/test-db")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGINT/SIGTERM.
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		log.Println("shutdown signal received")
		cancel()
	}()

	client, err := spanner.NewClient(ctx, spannerDB)
	if err != nil {
		log.Fatalf("spanner.NewClient: %v", err)
	}
	defer client.Close()

	clk := clock.RealClock{}
	prodRepo := repo.NewProductRepo()
	outboxRepo := repo.NewOutboxRepo()
	cm := committer.NewAdapter(client)
	readModel := queries.NewSpannerReadModel(client)

	// CQRS wiring
	cmds := grpcproduct.Commands{
		Create:   create_product.NewInteractor(prodRepo, outboxRepo, cm, clk),
		Update:   update_product.NewInteractor(prodRepo, outboxRepo, cm, readModel, clk),
		Activate: activate_product.NewInteractor(prodRepo, outboxRepo, cm, readModel, clk),
		ApplyDis: apply_discount.NewInteractor(prodRepo, outboxRepo, cm, readModel, clk),
	}
	qrys := grpcproduct.Queries{
		Get:  get_product.NewHandler(readModel),
		List: list_products.NewHandler(readModel),
	}
	h := grpcproduct.NewHandler(cmds, qrys)

	// gRPC server
	srv := grpc.NewServer()
	productv1.RegisterProductServiceServer(srv, h)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := srv.Serve(lis); err != nil {
			log.Printf("grpc serve: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	stopped := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		srv.Stop()
	}

	log.Println("server stopped")
}

func env(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
