# Руководство пользователя Multica MCP

Это руководство описывает сценарии использования MCP-сервера Multica с локальными агентами.

Сервер ориентирован на **Multica REST API v0.4.11**.

## Установка

### Установка через Go

Требуется Go 1.25+ и каталог `GOBIN` в `PATH` (часто `~/go/bin`):

```bash
go install github.com/strider2038/multica-mcp@latest
```

Бинарный файл: **`multica-mcp`** (обычно `$(go env GOPATH)/bin/multica-mcp`).

### Сборка из исходников

```bash
git clone https://github.com/strider2038/multica-mcp.git
cd multica-mcp
make build
```

Бинарный файл появится в `bin/multica-mcp`.

### Настройка окружения

```bash
export MULTICA_BASE_URL=https://multica.ai
export MULTICA_TOKEN=mul_ваш_токен
export MULTICA_WORKSPACE_ID=ws_abc123  # необязательно, если у вас одно пространство
# или: export MULTICA_WORKSPACE_SLUG=my-team  # вместо ID; если заданы оба, используется slug
```

Получить токен: Multica → Настройки → Personal Access Tokens → Создать.

## Подключение к агенту

### Claude Code

Добавьте в `.claude/settings.json`:

```json
{
  "mcpServers": {
    "multica": {
      "command": "/путь/к/multica-mcp",
      "env": {
        "MULTICA_BASE_URL": "https://multica.ai",
        "MULTICA_TOKEN": "mul_ваш_токен",
        "MULTICA_WORKSPACE_ID": "ws_abc123"
      }
    }
  }
}
```

### OpenCode

Добавьте в `opencode.json`:

```json
{
  "mcp": {
    "multica": {
      "command": "multica-mcp",
      "env": {
        "MULTICA_BASE_URL": "https://multica.ai",
        "MULTICA_TOKEN": "mul_ваш_токен"
      }
    }
  }
}
```

## Сценарии использования

### 1. Просмотр проектов

Скажите агенту:
> Покажи мои проекты

Агент вызовет `multica_list_projects` и покажет список:
```
- Backend API (12 задач, 3 завершено)
- Frontend Redesign (8 задач, 1 завершено)
```

### 2. Просмотр активных задач

> Покажи активные задачи в проекте Backend API

Агент вызовет `multica_list_tasks` с фильтром по проекту и статусу.

### 3. Поиск задач

> Найди задачи про авторизацию

Агент вызовет `multica_search_tasks` с поисковым запросом. Поиск идёт по заголовкам, описаниям и комментариям.

### 4. Создание задачи

> Создай задачу в проекте Backend API: «Добавить пагинацию» с описанием «Реализовать cursor-based пагинацию»

Агент вызовет `multica_create_task`:
```json
{
  "project_id": "p1",
  "title": "Добавить пагинацию",
  "description": "Реализовать cursor-based пагинацию"
}
```

### 5. Разбиение задачи на подзадачи

Сначала спланируйте:
> Спланируй разбивку задачи «Реализовать OAuth2 авторизацию»

Агент вызовет `multica_plan_task_breakdown` — создаст план без сохранения:
```
1. Анализ и требования
2. Реализация
3. Тестирование
4. Документация
```

Затем создайте:
> Создай задачу с подзадачами в проекте Backend

Агент вызовет `multica_create_task_with_subtasks`.

### 6. Добавление комментария

> Добавь комментарий к задаче MUL-42: «Начал работу, жду ревью»

Агент вызовет `multica_add_comment`.

### 7. Назначение задачи агенту

> Назначь задачу MUL-42 на агента Claude

Агент вызовет `multica_list_agents` для получения ID, затем `multica_assign_task`.

### 8. Обновление статуса

> Поставь задаче MUL-42 статус in_progress

Агент вызовет `multica_update_task` со статусом `in_progress`.

### 9. Просмотр деталей задачи

> Покажи задачу MUL-42

Агент вызовет `multica_get_task` — вернёт описание, статус, комментарии и подзадачи.

## Дополнительные возможности

### Dry-run режим

Все create/update инструменты поддерживают `dry_run: true` — операция валидируется без реального создания:

```json
{
  "project_id": "p1",
  "title": "Тестовая задача",
  "description": "Описание",
  "dry_run": true
}
```

### Read-only режим

Установите `MULTICA_READ_ONLY=true` — все write-инструменты будут отключены:
- `multica_create_task`
- `multica_create_subtask`
- `multica_update_task`
- `multica_add_comment`
- `multica_assign_task`
- `multica_create_task_with_subtasks`

### Валидация статусов

Сервер не допустит невалидный статус. Допустимые значения:
- `backlog` — не запланировано
- `todo` — запланировано
- `in_progress` — в работе
- `in_review` — на ревью
- `done` — завершено
- `blocked` — заблокировано
- `cancelled` — отменено

### Приоритеты

- `none` — не указан (по умолчанию)
- `urgent` — срочно
- `high` — высокий
- `medium` — средний
- `low` — низкий

## Трансмиссия

### stdio (по умолчанию)

Агент запускает сервер как дочерний процесс и общается через stdin/stdout.

### HTTP

```bash
MCP_TRANSPORT=http MCP_HTTP_PORT=9090 ./bin/multica-mcp
```

Сервер будет доступен на `http://localhost:9090/mcp`.

## Размещение на VPS (self-hosted)

Для запуска на удалённом сервере включите аутентификацию через API-ключ:

```bash
MCP_TRANSPORT=http \
MCP_HTTP_PORT=8080 \
MCP_API_KEY=ваш-секретный-ключ \
./bin/multica-mcp
```

Клиенты должны передавать заголовок `Authorization: Bearer ваш-секретный-ключ` с каждым запросом. Без `MCP_API_KEY` аутентификация отключена — подходит только для локального использования через stdio.

### Reverse proxy (Caddy)

```Caddyfile
multica-mcp.example.com {
    reverse_proxy localhost:8080
}
```

### Подключение удалённого агента

```json
{
  "mcpServers": {
    "multica": {
      "url": "https://multica-mcp.example.com/mcp",
      "headers": {
        "Authorization": "Bearer ваш-секретный-ключ"
      }
    }
  }
}
```

## Устранение неполадок

### «configuration error: MULTICA_BASE_URL is required»

Установите переменную окружения `MULTICA_BASE_URL`.

### «configuration error: MULTICA_TOKEN is required»

Создайте PAT в Multica: Настройки → Personal Access Tokens → Создать.

### «multiple workspaces found»

Установите `MULTICA_WORKSPACE_ID` или `MULTICA_WORKSPACE_SLUG` вручную. Узнать ID и slug можно через CLI: `multica workspace list`.

### 401 Unauthorized

Проверьте токен. Возможно, он был отозван — создайте новый.

### Агент не видит инструменты

Проверьте, что сервер запускается без ошибок. Запустите вручную с `LOG_LEVEL=debug`.

## Архитектура

```
Запрос от агента
    ↓
MCP Transport (stdio/http)
    ↓
MCP Tool Handlers (internal/mcp)
    ↓
Use Cases (internal/app)
    ↓
Multica HTTP Client (internal/multica)
    ↓
Multica API
```

Бизнес-логика не зависит от MCP SDK — только слой `internal/mcp` знает о протоколе MCP.
