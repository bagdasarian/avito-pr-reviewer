.PHONY: help build run test test-verbose test-coverage test-integration clean deps fmt vet lint docker-up docker-down docker-restart load-test-low load-test-medium load-test-high load-test-stress

APP_NAME=avito-pr-reviewer
BINARY_NAME=bin/$(APP_NAME)
MAIN_PATH=./cmd/app/main.go
DOCKER_COMPOSE=docker-compose

help:
	@echo "  build                Собрать приложение"
	@echo "  run                  Запустить приложение локально"
	@echo "  deps                 Установить зависимости"
	@echo "  test                 Запустить тесты"
	@echo "  test-verbose         Запустить тесты с подробным выводом"
	@echo "  test-coverage        Запустить тесты с покрытием кода"
	@echo "  test-integration     Запустить интеграционные тесты"
	@echo "  fmt                  Форматировать код"
	@echo "  clean                Очистить артефакты сборки"
	@echo "  docker-up            Запустить сервисы через docker-compose"
	@echo "  docker-down          Остановить docker-compose"
	@echo "  docker-restart       Перезапустить docker-compose"
	@echo "  docker-logs          Показать логи docker-compose"
	@echo "  docker-build         Собрать Docker образ"
	@echo "  load-test-low        Запустить нагрузочный тест (low)"
	@echo "  load-test-medium     Запустить нагрузочный тест (medium)"
	@echo "  load-test-high       Запустить нагрузочный тест (high)"
	@echo "  load-test-stress     Запустить нагрузочный тест (stress)"
	@echo "  all                  Выполнить полный цикл: очистка, зависимости, форматирование, тесты, сборка"

build:
	@mkdir -p bin
	@go build -o $(BINARY_NAME) $(MAIN_PATH)

run:
	@go run $(MAIN_PATH)

deps:
	@go mod download
	@go mod tidy

test:
	@go test ./...

test-verbose:
	@go test -v ./...

test-coverage:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

test-integration:
	@go test -tags=integration -v ./test/integration/...

fmt:
	@go fmt ./...

clean:
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean

docker-up:
	@$(DOCKER_COMPOSE) up -d

docker-down: 
	@$(DOCKER_COMPOSE) down

docker-restart: docker-down docker-up 

docker-logs: 
	@$(DOCKER_COMPOSE) logs -f

docker-build:
	@$(DOCKER_COMPOSE) build

load-test-low:
	@k6 run load_test/load_test_low.js

load-test-medium:
	@k6 run load_test/load_test_medium.js

load-test-high:
	@k6 run load_test/load_test_high.js

load-test-stress:
	@k6 run load_test/load_test_stress.js

all: clean deps fmt test build
