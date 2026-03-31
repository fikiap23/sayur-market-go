## Project Overview

Micro Sayur is a Go-based microservices project for managing an online vegetable store. It consists of multiple services:

- **user-service** (Go 1.25)
- **product-service** (Go 1.24)
- **notification-service** (Go 1.21)
- **order-service** (implied)
- **payment-service** (implied)

## Build, Lint, and Test Commands

### Running Services

```bash
# Start all services with Docker
make up

# Stop all Docker containers
make down

# View logs
make logs
```

### Go Module Management

```bash
# Run go mod tidy in all services
make mod-tidy

# Download dependencies
make mod-download

# Verify dependencies
make mod-verify

# Run all go mod commands
make mod-all
```

### Running Individual Services

Each service can be run directly with Go:

```bash
# user-service
cd user-service && go run main.go

# product-service
cd product-service && go run main.go

# notification-service
cd notification-service && go run main.go
```

### Testing

This codebase does not currently have unit tests (`*_test.go` files). When adding tests:

```bash
# Run all tests in a service
cd user-service && go test ./...

# Run a single test file
cd user-service && go test -v ./internal/core/service/user_service_test.go

# Run a specific test function
cd user-service && go test -v -run TestFunctionName ./...

# Run tests with coverage
cd user-service && go test -cover ./...
```

### Building

```bash
# Build individual services
cd user-service && go build -o bin/user-service main.go
cd product-service && go build -o bin/product-service main.go
cd notification-service && go build -o bin/notification-service main.go
```

## Code Style Guidelines

### Project Structure

Follow the hexagonal architecture pattern:

```
internal/
  adapter/          # Driving (handlers) and driven (repositories) adapters
    handler/        # HTTP handlers (Echo)
      request/     # Request DTOs
      response/    # Response DTOs
    repository/    # Database repositories
    message/       # RabbitMQ message adapters
  core/
    domain/
      entity/      # Domain entities
      model/       # Database models (GORM)
    service/       # Business logic
  middleware/      # Echo middleware
config/            # Configuration
utils/             # Utility packages
cmd/               # Command-line entry points
database/
  migrations/      # SQL migrations
  seeds/           # Database seeders
```

### Naming Conventions

- **Packages**: Use short, lowercase names (e.g., `repository`, `service`, `entity`)
- **Interfaces**: Use `Interface` suffix (e.g., `UserRepositoryInterface`, `UserServiceInterface`)
- **Structs**: Use camelCase with capital first letter (e.g., `userService`, `userHandler`)
- **Private fields**: Use camelCase (e.g., `db`, `repo`, `cfg`)
- **Database models**: Use singular (e.g., `User`, `Role`)
- **Tables**: Plural, snake_case in SQL migrations

### Imports

Group imports in the following order:

1. Standard library
2. External packages (github.com, etc.)
3. Internal packages

```go
import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "time"

    "user-service/config"
    "user-service/internal/adapter/repository"
    "user-service/internal/core/domain/entity"

    "github.com/google/uuid"
    "github.com/rs/zerolog"
    "gorm.io/gorm"
)
```

### Error Handling

- Use `errors.New()` for simple errors
- Use `fmt.Errorf()` with `%w` for wrapped errors
- Return error strings like "404", "401" for HTTP status mapping in handlers
- Log errors with appropriate context using zerolog or gommon/log

```go
// In repository
if errors.Is(err, gorm.ErrRecordNotFound) {
    err = errors.New("404")
    log.Errorf("[UserRepository-1] GetUserByID: %v", err)
    return nil, err
}

// In service
if err != nil {
    u.logger.Error().Err(err).Msg("hash password failed")
    return err
}

// In handler
if err.Error() == "404" {
    resp.Message = "Customer not found"
    return c.JSON(http.StatusNotFound, resp)
}
```

### Logging

- Use zerolog for structured logging in services
- Use gommon/log in handlers
- Include contextual information with log format: `[PackageName-LineNumber] FunctionName: message`

```go
// Service (zerolog)
u.logger.Error().Err(err).Msg("hash password failed")

// Handler (gommon)
log.Errorf("[UserHandler-1] SignIn: %v", err)
```

### Validation

Use go-playground/validator for request validation:

```go
if err = c.Validate(&req); err != nil {
    log.Errorf("[UserHandler-3] UpdateCustomer: %v", err)
    resp.Message = err.Error()
    return c.JSON(http.StatusBadRequest, resp)
}
```

### Database Transactions

Use GORM transactions for atomic operations:

```go
return u.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    txRepo := u.repo.WithTx(tx)

    if err := txRepo.CreateCustomer(ctx, req); err != nil {
        return err
    }

    if err := u.insertNotifOutbox(tx, userID, req.Email, msg, utils.NOTIF_EMAIL_CREATE_CUSTOMER, "Account Exists"); err != nil {
        return err
    }

    return nil
})
```

### Configuration

Use Viper for configuration management:

```go
type Config struct {
    App      App      `json:"app"`
    Psql     PsqlDB   `json:"psql"`
    RabbitMQ RabbitMQ `json:"rabbitmq"`
    Storage  Supabase `json:"storage"`
    Redis    Redis    `json:"redis"`
}
```

### HTTP Responses

Use consistent response structures:

```go
type DefaultResponse struct {
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}

type DefaultResponseWithPaginations struct {
    Message    string      `json:"message"`
    Data       interface{} `json:"data"`
    Pagination *Pagination `json:"pagination,omitempty"`
}
```

### Context Usage

Always pass context through the call chain:

```go
func (u *userService) GetUserByID(ctx context.Context, userID int64) (*entity.UserEntity, error)
func (u *userRepository) GetUserByID(ctx context.Context, userID int64) (*entity.UserEntity, error)
```

### Dependency Injection

Use constructor functions for dependency injection:

```go
func NewUserService(
    db *gorm.DB,
    repo repository.UserRepositoryInterface,
    cfg *config.Config,
    jwtService JwtServiceInterface,
    repoToken repository.VerificationTokenRepositoryInterface,
    outboxRepo repository.OutboxRepositoryInterface,
    logger zerolog.Logger,
) UserServiceInterface {
    return &userService{
        db:         db,
        repo:       repo,
        cfg:        cfg,
        jwtService: jwtService,
        repoToken:  repoToken,
        outboxRepo: outboxRepo,
        logger:     logger.With().Str("component", "user_service").Logger(),
    }
}
```

### Code Comments

Add comments for public functions and complex logic. Use doc comments for exported functions:

```go
// CreateCustomer creates a customer and writes a notification outbox event
// in the same transaction.
func (u *userService) CreateCustomer(ctx context.Context, req entity.UserEntity) error {
```

### Environment Configuration

Copy `.env.example` to `.env` for local development:

```bash
cp user-service/.env.example user-service/.env
```

### Docker Development

Use the provided `docker-compose.yml` for running all infrastructure services (PostgreSQL, RabbitMQ, Redis, Elasticsearch).

## Payment Service – Midtrans Webhook Setup

### Endpoint yang dipakai

- **Method**: `POST`
- **Path**: `/payments/webhook`
- **Service**: `payment-service` (default port `8084`)

Handler ini diregistrasi di `payment-service` lewat:

```go
e.POST("/payments/webhook", paymentHandler.MidtranswebHookHandler)
```

### Konfigurasi environment

Di `.env` / `.env.example` `payment-service`:

```env
MIDTRANS_SERVER_KEY=SB-Mid-server-xxxx        # ganti dengan server key kamu
MIDTRANS_ENVIRONMENT=0                        # 0 = sandbox, 1 = production
```

Pastikan `MIDTRANS_ENVIRONMENT` diset sesuai mode yang kamu pakai:

- `0` → Sandbox (testing)
- `1` → Production (real payment)

### Mendaftarkan webhook di Midtrans Dashboard

1. Jalankan `payment-service` (misalnya via `make up` atau `cd payment-service && go run main.go`).
2. Jika akses dari internet/public:
   - Gunakan tunneling seperti `ngrok http 8084`.
   - Catat URL publik yang diberikan ngrok, misalnya `https://abcd1234.ngrok.io`.
3. Buka **Midtrans Dashboard** → **Settings** → **Configuration**.
4. Isi **Payment Notification URL** dengan:

   - `https://abcd1234.ngrok.io/payments/webhook` (untuk lokal via ngrok), atau
   - `https://api.your-domain.com/payments/webhook` (untuk environment deploy).

5. Simpan konfigurasi lalu lakukan transaksi test di sandbox untuk memastikan:
   - Midtrans mengirim notification ke `/payments/webhook`.
   - Status payment di database dan downstream service (order-service) ikut ter-update.

