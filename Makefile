# Daftar semua service
SERVICES := user-service product-service order-service payment-service notification-service

# Target untuk menjalankan go mod tidy di semua service
.PHONY: mod-tidy
mod-tidy:
	@for service in $(SERVICES); do \
		echo "Running go mod tidy in $$service..."; \
		cd $$service && go mod tidy && cd ..; \
	done

# Target untuk menjalankan go mod download di semua service
.PHONY: mod-download
mod-download:
	@for service in $(SERVICES); do \
		echo "Running go mod download in $$service..."; \
		cd $$service && go mod download && cd ..; \
	done

# Target untuk menjalankan go mod verify di semua service
.PHONY: mod-verify
mod-verify:
	@for service in $(SERVICES); do \
		echo "Running go mod verify in $$service..."; \
		cd $$service && go mod verify && cd ..; \
	done

# Target untuk menjalankan semua perintah go mod
.PHONY: mod-all
mod-all: mod-tidy mod-download mod-verify

# Target untuk menjalankan docker compose up
.PHONY: up
up:
	docker compose up -d

# Target untuk menjalankan docker compose down
.PHONY: down
down:
	docker compose down

# Target untuk melihat log semua container
.PHONY: logs
logs:
	docker compose logs -f

# Bantuan
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make mod-tidy     - Run go mod tidy in all services"
	@echo "  make mod-download - Run go mod download in all services"
	@echo "  make mod-verify   - Run go mod verify in all services"
	@echo "  make mod-all      - Run all go mod commands in all services"
	@echo "  make up           - Start all Docker containers with docker compose up -d"
	@echo "  make down         - Stop all Docker containers with docker compose down"
	@echo "  make logs         - View logs from all Docker containers with docker compose logs -f"