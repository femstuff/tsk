# TSK MVP Monorepo

Минимально рабочий сквозной MVP:

- `apps/backend-api` — Go backend API (PostgreSQL, файловое хранилище, Redis-кэш)
- `apps/admin-web` — React + TypeScript админ-панель
- `apps/mobile-app` — Expo / React Native мобильное приложение
- `infra` — Prometheus + Grafana

Legacy-сервисы `bot-service` и `transc-python-service` сохранены отдельно и не входят в основной Docker-стек автоматически.

## Что работает

### Backend API

Данные хранятся в **PostgreSQL**, файлы — во volume `backend-storage`. **Redis** (если задан `REDIS_URL`) кэширует списки шаблонов и Bitrix-задач/сделок на 30–60 секунд; сессии OAuth хранятся в PostgreSQL.

Основные сущности: шаблоны документов, заявки (jobs), сгенерированные и исходные файлы, события обработки, команды интеграций, OAuth-сессии Bitrix24.

Ключевые endpoints:

| Группа | Маршруты |
|--------|----------|
| Health / metrics | `GET /api/v1/health`, `GET /metrics` |
| Документы | `GET, POST /api/v1/document-templates`, jobs, source/generated documents |
| Mobile | `POST /api/v1/mobile/voice-requests`, Bitrix intent, tasks, deals, notifications |
| Bitrix OAuth | `GET /api/v1/mobile/bitrix/oauth/start`, callback, session |
| Admin | dashboard, processing-events, task-commands |

### Admin Web (`http://localhost:5173`)

Пять разделов:

1. **Обзор** — KPI, график заявок/API, последние авторизации Bitrix, последние события
2. **Bitrix24** — задачи (диаграмма + фильтр), OAuth-пользователи
3. **Документы** — вкладки: заявки, шаблоны, файлы, новая заявка
4. **Журнал** — события и команды backend
5. **Сервер** — health, метрики Prometheus, ссылки на Grafana

Прокси same-origin: `/api`, `/health`, `/metrics` → backend (nginx в Docker, Vite proxy локально).

### Mobile App

Главный экран (вкладки **Главная** / **Сделки** / **Документы** / **Ещё**):

- **Bitrix24 OAuth** — вход через иконку профиля; задачи, сделки, уведомления
- **Задачи** — список, карточка, смена статуса, **отдельный экран чата** с отправкой сообщений
- **Сделки** — список, смена стадии, редактирование полей; суммы с двумя знаками после запятой
- **Голос / текст Bitrix** — intent с подтверждением сделки при неоднозначности
- **Оценка (estimate)** — загрузка шаблона и генерация документа
- **Уведомления** — отдельный экран по колокольчику, «прочитать все»
- **Ещё** — статус сервера, журнал запросов

## Bitrix24

### Webhook (сервер)

`BITRIX_WEBHOOK_URL` в `.env` корня репозитория:

```
BITRIX_WEBHOOK_URL=https://<портал>.bitrix24.ru/rest/<user>/<token>/
```

Без URL webhook-вызовы не выполняются; команды сохраняются как recorded/pending.

### OAuth (мобильное приложение)

Пользователь авторизуется в приложении. Backend хранит access/refresh token и вызывает Bitrix от его имени.

Для **чата задач** и **уведомлений** нужны scopes `task`, `im` (и CRM для сделок).

Отправка сообщений в задачу (backend пробует по порядку):

1. `tasks.task.chat.message.send` — новая карточка задачи (REST v3, `/rest/api/...`)
2. `im.message.add` — чат задачи по `CHAT` из `tasks.task.get`
3. `task.commentitem.add` — legacy-комментарии (с `AUTHOR_ID` OAuth-пользователя)

### Голос → Whisper

В Docker поднимается `whisper-api` (`WHISPER_BASE_URL=http://whisper-api:8000`). Мобильное приложение отправляет аудио на backend; транскрипция и Bitrix-intent обрабатываются на сервере.

## Где что хранится

| Хранилище | Содержимое |
|-----------|------------|
| PostgreSQL | templates, jobs, documents, events, task_commands, bitrix_oauth_sessions |
| Volume `backend-storage` | файлы шаблонов, voice/uploads, generated docs |
| Redis (опционально) | кэш списков Bitrix и шаблонов |

## Локальный запуск

### Windows PowerShell

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dev-up.ps1
```

После изменений в `backend-api` или `admin-web` не используйте `-SkipBuild`.

Логи: `.\scripts\dev-logs.ps1` · Остановка: `.\scripts\dev-down.ps1`

### macOS / Linux

```bash
make build && make up
make logs    # логи
make down    # остановка
```

### Сервисы

| Сервис | URL |
|--------|-----|
| PostgreSQL | `postgres://tsk:tsk@localhost:5432/tsk` |
| Backend API | `http://localhost:8080/` |
| Health | `http://localhost:8080/api/v1/health` |
| Admin | `http://localhost:5173` |
| Whisper | `http://localhost:8000/transcribe` |
| Prometheus | `http://localhost:9090` |
| Grafana | `http://localhost:3000` (admin / admin) |

## Mobile app

```bash
cd apps/mobile-app
npm install
npm run start
```

По умолчанию API:

- Android emulator: `http://10.0.2.2:8080`
- iOS simulator / прочее: `http://localhost:8080`

Другой хост:

```powershell
$env:EXPO_PUBLIC_API_BASE_URL="http://192.168.1.50:8080"
npm run start
```

После изменений backend пересоберите образ:

```powershell
docker compose build backend-api
docker compose up -d backend-api
```

## Метрики

Backend экспортирует Prometheus-метрики: HTTP traffic, business requests, **заявки на документы** (не задачи Bitrix24), errors, uptime. Grafana dashboard: **TSK Backend Overview**.

Счётчики `tsk_document_jobs_by_status` и `tsk_document_jobs_total` синхронизируются с PostgreSQL каждые 30 секунд и при создании/смене статуса заявки. После деплоя перезапустите `backend-api` и обновите dashboard в Grafana (или `docker compose restart grafana`).

Admin polling помечается служебным заголовком и не попадает в business-метрики.

Полезные запросы:

- `up{job="backend-api"}`
- `sum(tsk_business_http_requests_total)`
- `sum by (status) (tsk_document_jobs_by_status)`

## Минимальная проверка

1. `.\scripts\dev-up.ps1` или `make up`
2. `http://localhost:5173` — обзор без ошибок
3. Mobile: OAuth Bitrix → задачи / сделки / чат
4. Голосовой или текстовый Bitrix-intent
5. Grafana: dashboard после нескольких API-запросов

## Legacy

```bash
make up-legacy
make logs-legacy
```

Отдельный profile, не мешает основному стеку.
