# Product Catalog Service

A production-grade Product Catalog Service implementing domain-driven design, CQRS, and the transactional outbox pattern using Google Cloud Spanner and gRPC.

## Overview

This service manages products and their pricing with proper domain-driven design and clean architecture principles. It demonstrates production patterns including:

- **Clean Architecture** with strict layer separation
- **Domain-Driven Design** with rich domain models and encapsulation
- **CQRS** separating command and query responsibilities
- **Transactional Outbox Pattern** for reliable event publishing
- **Precise Decimal Arithmetic** using math/big for money calculations
- **Golden Mutation Pattern** for atomic transactions with CommitPlan

## Technology Stack

- **Language:** Go 1.21+
- **Database:** Google Cloud Spanner (with emulator for local development)
- **Transport:** gRPC with Protocol Buffers
- **Transaction Management:** github.com/Vektor-AI/commitplan with Spanner driver
- **Decimal Precision:** math/big for money calculations
- **Testing:** Standard Go testing with testify assertions

## Architecture

### Layer Structure

```
┌─────────────────────────────────────────┐
│         gRPC Transport Layer            │
│  (Handlers, Proto Mapping, Validation)  │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│        Application Layer                │
│    (Use Cases, Queries, DTOs)           │
└─────────────────┬───────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│          Domain Layer                   │
│  (Aggregates, Value Objects, Events)    │
│         PURE BUSINESS LOGIC             │
└─────────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────┐
│      Infrastructure Layer               │
│  (Repositories, Database Models)        │
└─────────────────────────────────────────┘
```

### Domain Model

The core domain consists of:

- **Product Aggregate**: Encapsulates product identity, pricing, discounts, and status
- **Money Value Object**: Precise decimal representation using big.Rat
- **Discount Value Object**: Percentage-based discounts with validity periods
- **Pricing Calculator**: Domain service for computing effective prices
- **Domain Events**: ProductCreated, ProductUpdated, DiscountApplied, etc.

### Key Patterns

#### The Golden Mutation Pattern

Every write operation follows this atomic transaction pattern:

1. Load or create domain aggregate
2. Execute business logic (domain methods)
3. Build CommitPlan by collecting mutations from repositories
4. Add outbox events for domain events
5. Apply the entire plan atomically

This ensures consistency between state changes and event publishing.

#### CQRS Separation

- **Commands** (writes) flow through domain aggregates and enforce all business rules
- **Queries** (reads) bypass the domain and use optimized read models with DTOs
- This allows independent optimization of read and write paths

#### Repository Pattern

Repositories follow a critical constraint: they return Spanner mutations but never apply them. The use case layer is responsible for collecting all mutations into a CommitPlan and applying them atomically. This enables:

- Transactional consistency across multiple aggregate updates
- Atomic event publishing via outbox pattern
- Change tracking for optimized updates (only modified fields)

#### Change Tracking

Domain aggregates track which fields have been modified. Repositories consult this tracker to build minimal update mutations, reducing write contention and improving performance.

## Prerequisites

- Go 1.21 or later
- Docker + Docker Compose
- Protocol Buffers compiler (`protoc`) + Go plugins

## Quick Start

### 1. Start the Spanner Emulator

```bash
docker compose up -d
```

Then make sure your shell can reach it from the host:

```bash
export SPANNER_EMULATOR_HOST=localhost:9010
```

**Windows PowerShell:**

```powershell
$env:SPANNER_EMULATOR_HOST = "localhost:9010"
```

### 2. Run Database Migrations

Migrations are applied via a small Go helper that calls Spanner's Admin API against the emulator.

```bash
make migrate
```

This applies `migrations/001_initial_schema.sql` to the database referenced by `SPANNER_DATABASE` (defaults are in the `Makefile`).

### 3. Generate Protocol Buffers

Install generators:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Ensure `%GOPATH%\bin` is on your PATH (Windows) or `$(go env GOPATH)/bin` (macOS/Linux), then:

```bash
protoc -I proto `
--go_out=proto --go_opt=paths=source_relative `
--go-grpc_out=proto --go-grpc_opt=paths=source_relative `
proto/product/v1/product_service.proto
```
```bash
make proto
```

### 4. Run Tests

```bash
make test
```

The E2E tests run against the emulator and create a unique database per `go test` run (so they don't fight with each other).

### 5. Start the gRPC Server

```bash
make run
```

The server starts on `:50051` by default.

## Development Workflow

Run `make help` to see available targets. The required ones for the assignment are:

- `make migrate`
- `make test`
- `make run`

## Testing Strategy

### Unit Tests

Domain logic is tested in isolation without any external dependencies:

- Money value object arithmetic
- Discount validation logic
- Pricing calculator computations
- Business rule enforcement (e.g., cannot apply discount to inactive product)

### End-to-End Tests

E2E tests exercise the full stack using a real Spanner emulator connection:

- Product creation flow with event publishing
- Product updates with change tracking
- Discount application with price calculation
- Status transitions (activate/deactivate)
- Business rule validation returning proper domain errors
- Outbox event verification

The E2E tests validate the entire golden mutation pattern including transaction atomicity and event capture.

## Project Structure

```
product-catalog-service/
├── cmd/server/main.go              # Service entry point
├── internal/
│   ├── app/product/                # Product bounded context
│   │   ├── domain/                 # Pure business logic
│   │   │   ├── product.go          # Product aggregate
│   │   │   ├── discount.go         # Discount value object
│   │   │   ├── money.go            # Money value object
│   │   │   ├── domain_events.go    # Event definitions
│   │   │   ├── change_tracker.go
│   │   │   └── services/
│   │   │       └── pricing_calculator.go
│   │   ├── usecases/               # Application commands
│   │   │   ├── create_product/
│   │   │   ├── update_product/
│   │   │   ├── apply_discount/
│   │   │   └── activate_product/
│   │   ├── queries/                # Read operations
│   │   │   ├── get_product/
│   │   │   └── list_products/
│   │   ├── contracts/              # Interfaces
│   │   └── repo/                   # Spanner implementation
│   ├── models/                     # Database models
│   │   ├── m_product/
│   │   └── m_outbox/
│   ├── transport/grpc/             # gRPC handlers
│   └── pkg/                        # Shared utilities
├── proto/product/v1/               # gRPC API definitions
├── migrations/                     # Database schema
├── tests/e2e/                      # End-to-end tests
└── docker-compose.yml              # Spanner emulator
```

## Design Decisions and Trade-offs

### Domain Purity

**Decision:** The domain layer has zero external dependencies—no context.Context, no database imports, no proto definitions.

**Rationale:** This enforces true separation of business logic from infrastructure concerns, making the domain highly testable and portable. Business rules can be validated in microseconds without any I/O.

**Trade-off:** Requires mapping between layers (domain ↔ proto, domain ↔ database), which adds boilerplate. However, this explicit mapping prevents domain pollution and makes changes to infrastructure independent of business logic.

### big.Rat for Money

**Decision:** Use Go's math/big.Rat for all money calculations instead of floating-point or fixed-point decimal types.

**Rationale:** Provides arbitrary precision rational arithmetic, eliminating rounding errors in percentage-based discount calculations. Critical for financial correctness.

**Trade-off:** More verbose than float64 and requires careful numerator/denominator management. Storage in Spanner requires splitting into two INT64 columns. Performance is slightly lower, but correctness takes precedence.

### Repository Returns Mutations

**Decision:** Repositories return `*spanner.Mutation` objects but never apply them. Use cases collect all mutations into a CommitPlan.

**Rationale:** Enables atomic multi-aggregate transactions and transactional outbox pattern. All state changes and event publishing happen in a single Spanner transaction.

**Trade-off:** Less conventional than repositories that directly execute updates. Requires discipline to never call Apply() inside repositories. Benefits greatly outweigh the learning curve.

### Change Tracking Over Optimistic Locking

**Decision:** Use change tracking to generate minimal UPDATE mutations rather than full-row updates. Optimistic locking not implemented.

**Rationale:** Reduces write contention in Spanner by only updating modified columns. For this domain, last-write-wins is acceptable—products don't have complex concurrent modification requirements.

**Trade-off:** Concurrent updates to the same product could silently overwrite each other. For more critical domains, add a version field and check it in the repository. Omitted here for simplicity, but easy to add.

### CQRS Without Event Sourcing

**Decision:** Separate commands and queries, but persist current state rather than event streams.

**Rationale:** CQRS provides clear separation of concerns and optimization opportunities without the complexity of event sourcing. Domain events are captured for integration purposes, not as the source of truth.

**Trade-off:** Cannot reconstruct historical state or replay events to rebuild projections. For this catalog service, current state persistence is sufficient. Event sourcing could be added later if audit trails become critical.

### Outbox Without Background Processor

**Decision:** Store events in outbox table but don't implement the background processor that publishes them.

**Rationale:** Test scope focuses on the write-side patterns and transactional consistency. The processor is a separate concern (polling outbox, publishing to Pub/Sub, marking processed).

**Trade-off:** Events accumulate in the database but aren't published. In production, a separate worker would poll the outbox and publish events. The hard part (atomic event capture) is implemented here.

### Spanner Over PostgreSQL

**Decision:** Use Google Cloud Spanner despite its operational complexity.

**Rationale:** Demonstrates familiarity with distributed databases and cloud-native infrastructure. Spanner's external consistency guarantees and horizontal scalability align with production requirements.

**Trade-off:** Local development requires running the emulator, and schema migrations are DDL-based (no rollback). For a simpler test, PostgreSQL would suffice, but Spanner better represents real production environments.

### Minimal Use Case Interfaces

**Decision:** Use cases are concrete structs, not interface-based. No ports/adapters indirection.

**Rationale:** Interfaces are defined at repository boundaries where abstraction provides value (testing, swapping implementations). Use case interfaces add complexity without clear benefit in Go.

**Trade-off:** Slightly harder to mock use cases in tests. For this service, testing at the E2E layer (real database) is preferred anyway.

## API Overview

The gRPC API provides the following operations:

### Commands (Write Operations)

- `CreateProduct` - Create a new product with base pricing
- `UpdateProduct` - Modify product name, description, or category
- `ActivateProduct` - Enable a product for sale
- `DeactivateProduct` - Disable a product
- `ApplyDiscount` - Add percentage-based discount with date range
- `RemoveDiscount` - Remove active discount

### Queries (Read Operations)

- `GetProduct` - Retrieve product with current effective price
- `ListProducts` - List active products with pagination and category filtering

All commands publish domain events to the outbox table for downstream integration.

## Environment Variables

```bash
# Emulator endpoint (required)
SPANNER_EMULATOR_HOST=localhost:9010

# Full database resource name used by server + migrate tool
SPANNER_DATABASE=projects/test-project/instances/emulator-instance/databases/test-db

# Server
GRPC_PORT=50051
```

## Troubleshooting

### Emulator Won't Start

Check if port 9010 is already in use:

```bash
lsof -i :9010
```

View emulator logs:

```bash
make docker-logs
```

### Migration Fails

Ensure the emulator is running and instance/database exist:

```bash
make docker-up
gcloud spanner databases list --instance=test-instance --project=test-project
```

### Tests Fail With Connection Errors

The E2E tests automatically start the emulator, but if you see connection errors, manually verify:

```bash
docker-compose ps
```

If the container is not running:

```bash
make docker-up
```

## References

- [Vektor-AI CommitPlan](https://github.com/Vektor-AI/commitplan)
- [Google Cloud Spanner Documentation](https://cloud.google.com/spanner/docs)
- [gRPC Go Quickstart](https://grpc.io/docs/languages/go/quickstart/)
- [Domain-Driven Design](https://martinfowler.com/bliki/DomainDrivenDesign.html)
- [Transactional Outbox Pattern](https://microservices.io/patterns/data/transactional-outbox.html)
