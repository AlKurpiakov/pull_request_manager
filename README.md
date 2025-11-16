# PR Manager Service

Микросервис для автоматического управления процессом код-ревью в командах разработки. Сервис автоматически назначает ревьюверов для новых Pull Requests, предоставляет возможности переназначения и управления жизненным циклом PR.

## Возможности

### Основной функционал
- **Управление командами** - создание команд разработки
- **Управление пользователями** - регистрация участников команд с возможностью активации/деактивации
- **Автоматическое назначение ревьюверов** - система случайным образом выбирает 2 активных ревьювера из той же команды, что и автор PR
- **Переназначение ревьюверов** - замена одного ревьювера на другого активного участника команды
- **Управление PR** - создание, просмотр и слияние Pull Requests
- **Статистика** - отслеживание общего количества назначений

### Бизнес-логика
- Автоматически исключает автора PR из списка возможных ревьюверов
- Назначает только активных пользователей
- Не позволяет переназначать ревьюверов в уже смерженных PR
- Обеспечивает что ревьюверы принадлежат той же команде, что и автор
- При переназначении исключает уже назначенных ревьюверов из кандидатов

## Технологии

- **Go 1.21** - основной язык разработки
- **PostgreSQL 15** - реляционная база данных
- **Chi Router** - легковесный HTTP роутер
- **Docker & Docker Compose** - контейнеризация и оркестрация
- **pgx** - высокопроизводительный драйвер PostgreSQL
- **slog** - структурированное логирование

## Быстрый старт

### Способ 1: Docker Compose (рекомендуемый)

```bash
# Клонирование репозитория
git clone <repository-url>
cd prmanager

# Запуск всех сервисов
docker-compose up --build

# Приложение будет доступно на http://localhost:8080
# База данных на localhost:5432
```

### Способ 2: Локальная разработка

```bash
# Установка зависимостей
go mod download

# Запуск только базы данных
docker-compose up db -d

# Запуск приложения
go run main.go

# Или сборка и запуск
go build -o pr-manager ./cmd/pr-manager
./pr-manager
```

## Конфигурация
### Переменные окружения

Создайте файл `.env` на основе `.env.example`:

```env
# Database
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=pr_manager
DB_HOST=localhost
DB_PORT=5432
DB_CONN=postgres://postgres:postgres@localhost:5432/pr_manager?sslmode=disable

# Server
PORT=8080
READ_TIMEOUT=10
WRITE_TIMEOUT=10
IDLE_TIMEOUT=30
```

### Структура базы данных

Миграции выполняются автоматически при старте приложения:

```sql
-- Teams table
CREATE TABLE teams (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Users table  
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    team_id INT REFERENCES teams(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- PRs table
CREATE TABLE prs (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    author_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'OPEN',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- PR reviewers junction table
CREATE TABLE pr_reviewers (
    pr_id INT NOT NULL REFERENCES prs(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    PRIMARY KEY(pr_id, user_id)
);
```

## API Reference

### Команды

#### Создание команды
```http
POST /teams
Content-Type: application/json

{
    "name": "Backend Team"
}
```

Response:
```json
{
    "id": 1,
    "name": "Backend Team",
    "created_at": "2024-01-15T10:30:00Z"
}
```

#### Создание пользователя в команде
```http
POST /teams/{team_id}/users
Content-Type: application/json

{
    "name": "John Developer",
    "is_active": true
}
```

### Pull Requests

#### Создание PR
```http
POST /prs
Content-Type: application/json

{
    "title": "Implement new authentication",
    "author_id": 1
}
```

Response включает автоматически назначенных ревьюверов:
```json
{
    "id": 1,
    "title": "Implement new authentication",
    "author_id": 1,
    "status": "OPEN",
    "created_at": "2024-01-15T10:35:00Z",
    "reviewers": [
        {
            "id": 2,
            "team_id": 1,
            "name": "Alice Reviewer",
            "is_active": true,
            "created_at": "2024-01-15T10:31:00Z"
        },
        {
            "id": 3, 
            "team_id": 1,
            "name": "Bob Reviewer",
            "is_active": true,
            "created_at": "2024-01-15T10:32:00Z"
        }
    ]
}
```

#### Переназначение ревьювера
```http
POST /prs/{pr_id}/reassign
Content-Type: application/json

{
    "old_user_id": 2
}
```

#### Слияние PR
```http
POST /prs/{pr_id}/merge
```

### Запросы

#### Получение PR назначенных пользователю
```http
GET /users/{user_id}/prs
```

#### Статистика назначений
```http
GET /stats
```

Response:
```json
{
    "total_assignments": 42
}
```

## Тестирование

### Запуск тестов

```bash
make test
```

## Команды Makefile

```bash
make build          # Собрать приложение
make up            # Запустить в Docker
make down          # Остановить контейнеры
make test          # Запустить все тесты
make test-unit     # Только unit-тесты  
make test-integration # Только интеграционные тесты
make fmt           # Форматировать код
make lint          # Проверить код линтером
make clean         # Очистить проект
make logs          # Просмотр логов приложения
```

## Логирование

Приложение использует структурированное логирование через `slog`:

```json
{
    "time": "2024-01-15T10:30:00Z",
    "level": "INFO",
    "msg": "PR created successfully",
    "pr_id": 1,
    "reviewers_count": 2,
    "reviewer_ids": [2, 3]
}
```

## Мониторинг и диагностика

### Health Checks
- База данных: автоматическая проверка подключения при старте
- HTTP сервер: стандартные таймауты и обработка ошибок

### Метрики
- Количество назначений ревьюверов (`/stats` endpoint)
- Детальное логирование всех операций

## Обработка ошибок

Сервис возвращает структурированные ошибки:

```json
{
    "error": {
        "code": "NOT_FOUND",
        "message": "author not found"
    }
}
```

### Коды ошибок
- `BAD_REQUEST` - невалидные входные данные
- `NOT_FOUND` - ресурс не найден
- `INTERNAL_ERROR` - внутренняя ошибка сервера
- `PR_MERGED` - попытка изменить смерженный PR
- `NO_CANDIDATE` - нет доступных кандидатов для переназначения
- `NOT_ASSIGNED` - ревьювер не назначен на PR

##Безопасность

- Валидация всех входных данных
- Проверка прав доступа на уровне бизнес-логики
- SQL injection protection через параметризованные запросы
- Обработка edge cases (неактивные пользователи, отсутствующие команды)


## Разработка

### Code Style
```bash
# Автоматическое форматирование
make fmt

# Статический анализ
make lint
```

### Git Workflow
1. Создать feature branch
2. Написать тесты для новой функциональности
3. Реализовать функциональность
4. Убедиться что все тесты проходят
5. Создать Pull Request

---


**Примечание**: Для работы в production рекомендуется добавить аутентификацию, авторизацию и дополнительные механизмы безопасности.
