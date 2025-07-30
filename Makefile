# Go API Proxy Docker Management

.PHONY: help build up down logs clean dev prod

# Default target
help:
	@echo "Available commands:"
	@echo "  build     - Build Docker image"
	@echo "  up        - Start services (production)"
	@echo "  down      - Stop services"
	@echo "  logs      - View logs"
	@echo "  clean     - Clean up containers and images"
	@echo "  dev       - Start development environment"
	@echo "  prod      - Start production environment"
	@echo "  test      - Run health check test"

# Build Docker image
build:
	docker-compose build

# Start services (default/production)
up:
	docker-compose up -d

# Stop services
down:
	docker-compose down

# View logs
logs:
	docker-compose logs -f

# Clean up
clean:
	docker-compose down -v --rmi all --remove-orphans

# Development environment
dev:
	docker-compose -f docker-compose.dev.yml up -d

# Production environment
prod:
	docker-compose -f docker-compose.prod.yml up -d

# Stop development environment
dev-down:
	docker-compose -f docker-compose.dev.yml down

# Stop production environment
prod-down:
	docker-compose -f docker-compose.prod.yml down

# View development logs
dev-logs:
	docker-compose -f docker-compose.dev.yml logs -f

# View production logs
prod-logs:
	docker-compose -f docker-compose.prod.yml logs -f

# Health check test
test:
	@echo "Testing health endpoint..."
	@curl -f http://localhost/health || echo "Health check failed"

# Rebuild and restart
restart: down build up

# Development restart
dev-restart:
	docker-compose -f docker-compose.dev.yml down
	docker-compose -f docker-compose.dev.yml build
	docker-compose -f docker-compose.dev.yml up -d