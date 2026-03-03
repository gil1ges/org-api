DC := docker compose
GOCACHE ?= /tmp/gocache
GO_TEST_FLAGS ?= -count=1

.PHONY: test test-go test-vet up down reset health

test: test-go test-vet

test-go:
	mkdir -p $(GOCACHE)
	GOCACHE=$(GOCACHE) go test $(GO_TEST_FLAGS) ./...

test-vet:
	mkdir -p $(GOCACHE)
	GOCACHE=$(GOCACHE) go vet ./...

up:
	$(DC) up --build -d

down:
	$(DC) down

reset:
	$(DC) down -v
	$(DC) up --build -d

health:
	$(DC) exec -T app curl -i -sS http://127.0.0.1:8080/health
