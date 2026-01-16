
# Shortlink + Click Analytics (Go)

Сервис сокращения ссылок на Go с асинхронным сбором кликов и базовой статистикой.

> Цель проекта: показать backend-навыки (REST API, PostgreSQL, Docker, конкурентность в Go через worker pool + channels, корректное завершение через context).

---

## Возможности
- Создание коротких ссылок через REST API
- Редирект по короткому коду
- Сбор кликов **асинхронно** (очередь на `channel` + `worker pool`)
- Статистика по ссылке: `total_clicks`
- Хранение данных в PostgreSQL
- Миграции БД через `goose`
- Конфигурация через переменные окружения
- Graceful shutdown (остановка воркеров через `context`, дожим батча)
- Тесты (unit и/или integration) *(если есть — оставь этот пункт)*

---

## Стек
- Go
- HTTP: стандартный `net/http`
- PostgreSQL
- Docker + Docker Compose
- Миграции: `goose`

---

## Архитектура (коротко)

**Почему клики пишутся асинхронно?**  
Чтобы редирект был быстрым и не зависел от записи в БД на “горячем пути”.

Поток данных:



Client -> API
POST /links             -> сохраняет ссылку в Postgres
GET  /{code}            -> читает original_url -> редиректит
-> отправляет click event в channel

Click workers (N goroutines) читают события из channel
-> батчат (batch size / flush interval)
-> вставляют пачкой в Postgres

GET /links/{code}/stats   -> читает агрегаты из Postgres


<details>
  <summary><b>Структура проекта</b></summary>

  Пример (адаптируй под свой репозиторий):



cmd/
api/                # точка входа (main.go)
internal/
config/             # конфиг + env
http/
handler/          # HTTP handlers
middleware/       # middleware (логирование, recover, etc.)
service/            # бизнес-логика
repository/         # работа с БД
analytics/          # очередь кликов + воркеры
migrations/           # миграции PostgreSQL (goose)
docker-compose.yml
Dockerfile
Makefile              # (опционально)


</details>

 ```
## API

### 1) Создать короткую ссылку
`POST /links`

```bash
curl -X POST http://localhost:8080/links \
-H "Content-Type: application/json" \
-d '{"url":"https://example.com"}'
````

Пример ответа:

```json
{
  "code": "abc123",
  "short_url": "http://localhost:8080/abc123"
}
```

---

### 2) Редирект по коду

`GET /{code}`

```bash
curl -i http://localhost:8080/abc123
```

Ожидаемо: `301/302` + `Location: https://example.com`
Параллельно создаётся событие клика (асинхронно).

---

### 3) Статистика по ссылке

`GET /links/{code}/stats`

```bash
curl http://localhost:8080/links/abc123/stats
```

Пример ответа:

```json
{
  "code": "abc123",
  "total_clicks": 10
}
```

---

## Запуск (Quick start)

### Требования

* Docker
* Docker Compose
* `goose` (если миграции запускаются локально; при запуске через контейнер можно поставить внутрь образа)

### 1) Запуск зависимостей и сервиса

```bash
docker compose up --build
```

Сервис будет доступен по адресу:

* `http://localhost:8080`

### 2) Миграции (goose)

Если миграции **не применяются автоматически** при старте контейнера, запусти их вручную:

```bash
goose -dir ./migrations postgres "$DATABASE_URL" up
```

> Примечание: значение `DATABASE_URL` см. в разделе “Конфигурация”.

---

## Конфигурация (ENV)

| Переменная           |                                       Пример / Default | Описание                                      |
| -------------------- | -----------------------------------------------------: | --------------------------------------------- |
| APP_PORT             |                                                   8080 | Порт HTTP сервера                             |
| DATABASE_URL         | postgres://user:pass@localhost:5432/db?sslmode=disable | Строка подключения к Postgres                 |
| CLICK_WORKERS        |                                                      4 | Кол-во воркеров, обрабатывающих клики         |
| CLICK_BATCH_SIZE     |                                                    100 | Размер батча для записи кликов                |
| CLICK_FLUSH_INTERVAL |                                                     1s | Максимальное время ожидания перед flush батча |

---

```mermaid
flowchart TB
  Client[Client / Browser / API consumer]

  subgraph App[Go service: URL Shortener API]
    Router[HTTP Router + Middleware]
    Handler[Handlers]
    Service["Service layer (business logic)"]
    Repo["Repository (DB access)"]
    ClickProducer["Click Producer\n(puts events into channel)"]
    ClickChan[(chan ClickEvent)]
    Workers[Worker Pool\nN goroutines]
    Batcher["Batcher\n(size/time flush)"]
  end

  subgraph DB[PostgreSQL]
    Links[(links table)]
    Clicks[(clicks table)]
  end

  subgraph Infra[Infra]
    Compose[docker-compose]
    Migrations[goose migrations]
  end

  Client -->|POST /links| Router --> Service 
  Client -->|GET / code| Router --> Handler --> Service --> Repo 
  Handler -->|redirect 301/302| Client

  Handler -->|async click event| ClickProducer --> ClickChan
  ClickChan --> Workers --> Batcher --> Repo --> Clicks

  Compose --- App
  Compose --- DB
  Migrations --> DB
```

---

## Тестирование

```bash
go test ./...
```

Race detector (если проходит — большой плюс):

```bash
go test -race ./...
```

---

## Roadmap (что можно улучшить)

* Kafka mode для click events (опционально)
* Swagger/OpenAPI
* Авторизация (API key / JWT)
* Rate limiting на редирект
* Метрики (Prometheus) и трассировка

---

## Контакты

* Автор: Иван
* GitHub: `<ссылка>`
* Telegram: `<ссылка или @ник>`

```
::contentReference[oaicite:0]{index=0}
```
