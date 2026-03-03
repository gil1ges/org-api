# Org API

REST API для управления организационной структурой: подразделения, сотрудники, дерево подразделений.

## Что реализовано

- `net/http` + `gorm` + PostgreSQL
- Миграции через `goose` (выполняются при старте сервера)
- Docker + `docker-compose`
- Для `docker-compose` используется `network_mode: host` (Linux)
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

1. Скопировать этот репозиторий:

```bash
# SSH
git clone git@github.com:gil1ges/org-api.git
cd org-api

# или HTTPS
# git clone https://github.com/gil1ges/org-api.git
# cd org-api
```

2. Создать `.env` в корне проекта:

```bash
cat > .env << 'EOF'
APP_HOST=0.0.0.0
APP_PORT=8080
DB_HOST=localhost
DB_PORT=15432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=org_api
DB_SSLMODE=disable
EOF
```

3. Запустить сервисы:

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

# 1) Создать корневое подразделение
curl -i -X POST http://127.0.0.1:8080/departments/ \
  -H 'Content-Type: application/json' \
  -d '{"name":"Engineering"}'

# 2) Создать дочернее подразделение (id родителя = 1)
curl -i -X POST http://127.0.0.1:8080/departments/ \
  -H 'Content-Type: application/json' \
  -d '{"name":"Backend","parent_id":1}'

# 3) Добавить сотрудника в подразделение id=2
curl -i -X POST http://127.0.0.1:8080/departments/2/employees/ \
  -H 'Content-Type: application/json' \
  -d '{"full_name":"Ivan Ivanov","position":"Go Developer","hired_at":"2026-03-04"}'

# 4) Получить дерево подразделения id=1 вместе с сотрудниками
curl -i 'http://127.0.0.1:8080/departments/1?depth=2&include_employees=true'

# 5) Переименовать подразделение id=2
curl -i -X PATCH http://127.0.0.1:8080/departments/2 \
  -H 'Content-Type: application/json' \
  -d '{"name":"Backend Platform"}'

# 6) Удалить подразделение id=2 и перевести сотрудников в id=1
curl -i -X DELETE 'http://127.0.0.1:8080/departments/2?mode=reassign&reassign_to_department_id=1'

# 7) Каскадно удалить подразделение id=1
curl -i -X DELETE 'http://127.0.0.1:8080/departments/1?mode=cascade'

# 8) Негативный пример: сотрудник в несуществующее подразделение (должно быть 404)
curl -i -X POST http://127.0.0.1:8080/departments/9999/employees/ \
  -H 'Content-Type: application/json' \
  -d '{"full_name":"Ghost User","position":"N/A"}'

# 9) Негативный пример: некорректная глубина (должно быть 400)
curl -i 'http://127.0.0.1:8080/departments/1?depth=99&include_employees=true'
```

## Тесты

Остальные сценарии, валидации и пограничные кейсы проверяются автотестами:

- `make test`
- `go test ./...`
- `go test ./tests -run TestAPIFlow_CreateDepartmentEmployeeAndGetTree`

Ключевые тесты находятся в:

- `internal/service/service_test.go`
- `internal/handlers/handlers_test.go`
- `tests/api_smoke_test.go`

## Структура

- `cmd/server` - точка входа
- `internal/app` - сборка HTTP-приложения
- `internal/handlers` - HTTP-обработчики
- `internal/service` - бизнес-логика
- `internal/repository` - доступ к БД
- `internal/db` - подключение и миграции БД
- `migrations` - SQL-миграции goose
