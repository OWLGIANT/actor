.PHONY: all build run proto test clean docker-up docker-down docker-logs

# Variables
BINARY_DIR=bin
PROTO_DIR=protos
PROTO_OUTPUT_DIR=pkg/proto
GO=go
PROTOC=protoc
PROTOC_GO_PLUGIN=protoc-gen-go
PROTOC_GRPC_GO_PLUGIN=protoc-gen-go-grpc

all: proto build

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	@mkdir -p $(PROTO_OUTPUT_DIR)
	$(PROTOC) --proto_path=$(PROTO_DIR) \
		--go_out=$(PROTO_OUTPUT_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUTPUT_DIR) \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto

# Build all services
build: proto
	@echo "Building services..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build -o $(BINARY_DIR)/user-service ./cmd/user
	$(GO) build -o $(BINARY_DIR)/order-service ./cmd/order
	$(GO) build -o $(BINARY_DIR)/gateway ./cmd/gateway
	$(GO) build -o $(BINARY_DIR)/actor-service ./cmd/actor

# Run user service
run-user:
	$(GO) run ./cmd/user/main.go

# Run order service
run-order:
	$(GO) run ./cmd/order/main.go

# Run gateway
run-gateway:
	$(GO) run ./cmd/gateway/main.go

# Run actor service
run-actor:
	$(GO) run ./cmd/actor/main.go

# Run tests
test:
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@rm -f coverage.out coverage.html

# Docker commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Install dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Swagger generation
swagger:
	swag init -g cmd/gateway/main.go -o docs

# Lint
lint:
	golangci-lint run

# Format code
fmt:
	$(GO) fmt ./...

# Run database migrations
migrate-up:
	migrate -path migrations -database "mysql://root:123456@tcp(localhost:3306)/microshop" up

migrate-down:
	migrate -path migrations -database "mysql://root:123456@tcp(localhost:3306)/microshop" down

# Generate certificates for TLS
certs:
	@mkdir -p certs
	openssl req -x509 -newkey rsa:4096 -keyout certs/key.pem -out certs/cert.pem -days 365 -nodes -subj "/CN=localhost"

# Run all services (requires docker first)
run-all: docker-up
	@sleep 5
	@echo "Starting all services..."
	@$(MAKE) run-user &
	@$(MAKE) run-order &
	@$(MAKE) run-actor &
	@$(MAKE) run-gateway
