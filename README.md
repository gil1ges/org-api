# Org API

REST API для управления организационной структурой: подразделения, сотрудники, дерево подразделений.

## Что реализовано

- `net/http` + `gorm` + PostgreSQL
- Миграции через `goose` (выполняются при старте сервера)
- Docker + `docker-compose`
- Валидации из ТЗ:
  - `name`, `full_name`, `position` обязательны, длина `1..200`, trim
  - уникальность названия подразделения в пределах одного `parent_id` (case-insensitive)
  - запрет самоссылки и циклов в дереве
  - `depth` для дерева: `1..5` (по умолчанию `1`)
- Удаление подразделения:
  - `cascade`: удаляет подразделение, сотрудников и поддерево
  - `reassign`: удаляет только подразделение, сотрудников переносит в `reassign_to_department_id`, дочерние подразделения перепривязывает к родителю удаляемого
- Базовое логирование запросов
- Тесты (`go test ./...` + `tests/api_smoke_test.go`)

## Запуск

```bash
docker compose up --build -d
```

или через `Makefile`:

```bash
make test
make up
make health
```

- `make test` — прогнать `go test ./...` и `go vet ./...`
- `make up` — собрать и поднять `app + postgres`
- `make down` — остановить контейнеры
- `make reset` — полностью пересоздать стек с очисткой volume

## Проверка API

```bash
# GET /health
curl -i http://127.0.0.1:8080/health

# POST /departments/
curl -i -X POST http://127.0.0.1:8080/departments/ \
  -H 'Content-Type: application/json' \
  -d '{"name":"Engineering"}'

# GET /departments/{id}
curl -i 'http://127.0.0.1:8080/departments/1?depth=2&include_employees=true'
```

## Структура

- `cmd/server` - точка входа
- `internal/app` - сборка HTTP-приложения
- `internal/handlers` - HTTP-обработчики
- `internal/service` - бизнес-логика
- `internal/repository` - доступ к БД
- `internal/db` - подключение и миграции БД
- `migrations` - SQL-миграции goose
