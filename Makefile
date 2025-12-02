COMPOSE_FILE := docker-compose.yml

.PHONY: kafka-start kafka-stop kafka-status start

kafka-start:
	docker compose -f $(COMPOSE_FILE) up -d

start:
	JWT_SECRET=devsecret go run cmd/server/main.go

create-tocken:
	go run ./scripts/generate_jwt.go

kafka-stop:
	docker compose -f $(COMPOSE_FILE) down

kafka-status:
	docker compose -f $(COMPOSE_FILE) ps