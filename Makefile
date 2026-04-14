.PHONY: build up down clean logs migrate-up migrate-down

# Building image
build:
	docker-compose build --no-cache

# Start containers
up:
	docker-compose up -d

# Stop containers
down:
	docker-compose down

# Full clean
clean:
	docker-compose down -v --rmi all --remove-orphans

# Просмотр логов
logs:
	docker-compose logs -f