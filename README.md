# TSK MVP Monorepo

Этот репозиторий теперь содержит минимально рабочий сквозной MVP для:

- `apps/backend-api` - Go backend API
- `apps/admin-web` - React + TypeScript admin panel
- `apps/mobile-app` - Expo / React Native mobile client
- `infra` - Prometheus + Grafana

Legacy-сервисы `bot-service` и `transc-python-service` сохранены и не встроены
в новый MVP автоматически.

## Что уже работает

### Backend API

Новый backend хранит данные не в памяти, а в `PostgreSQL`, и использует файловое
storage под volume для шаблонов, входящих voice/source документов и сгенерированных
документов.

Поддерживаются:

- шаблоны документов
- jobs / requests
- generated documents
- source documents (например, voice recordings из mobile)
- processing events / operational log feed
- task commands / dispatch intents для Bitrix24 и email approval flow

Ключевые endpoints:

- `GET /api/v1/health`
- `GET, POST /api/v1/document-templates`
- `GET /api/v1/document-templates/{id}/download`
- `GET, POST /api/v1/document-jobs`
- `PATCH /api/v1/document-jobs/{id}/status`
- `POST /api/v1/mobile/voice-requests`
- `GET /api/v1/source-documents`
- `GET /api/v1/source-documents/{id}/download`
- `GET /api/v1/generated-documents`
- `GET /api/v1/generated-documents/{id}/download`
- `GET, POST /api/v1/task-commands`
- `GET /api/v1/processing-events`
- `GET /metrics`

### Admin Web

Админка показывает:

- статус backend
- шаблоны документов
- jobs / requests
- source documents
- generated documents
- task commands / dispatch intents
- operational log feed

Также из админки можно:

- загрузить новый template
- создать job вручную
- менять status job
- скачать template / source document / generated document

### Mobile App

Mobile app теперь закрывает минимальный пользовательский сценарий:

- загрузка шаблонов с backend
- запись voice note с микрофона
- создание document request с audio upload
- добавление manual notes / transcript text
- создание минимальной task command для `bitrix24` или `email_approval`
- просмотр собственных mobile requests, uploaded source docs и статусов task commands

## Что где хранится

### PostgreSQL

В БД сохраняются:

- `document_templates`
- `document_jobs`
- `generated_documents`
- `source_documents`
- `processing_events`
- `task_commands`

### File storage

Во volume backend storage сохраняются:

- файлы шаблонов
- voice recordings / source uploads из mobile
- generated document files

В Docker stack storage лежит в volume `backend-storage`.

## Интеграции

### Bitrix24

- URL входящего вебхука REST задаётся переменной **`BITRIX_WEBHOOK_URL`** (формат `https://<портал>.bitrix24.ru/rest/<user>/<token>/`).
- Для **Docker Compose** значение подставляется из файла **`.env` в корне репозитория** (файл в `.gitignore`). Пример строки: `BITRIX_WEBHOOK_URL=https://....bitrix24.ru/rest/1/xxxx/`
- В `docker-compose.yml` больше не зашит пустой вебхук: используется `BITRIX_WEBHOOK_URL: ${BITRIX_WEBHOOK_URL:-}` — без `.env` переменная пустая и запросы в Bitrix не уходят (это ожидаемо).
- если задан `BITRIX_WEBHOOK_URL`, backend вызывает REST-методы (`crm.deal.*`, `tasks.task.add` и т.д.)
- если webhook не задан, команда и intent сохраняются честно как recorded/pending flow

### Голос → Whisper → Bitrix (тест из админки)

В `docker-compose` поднимается сервис `whisper-api` (Python + Whisper). Backend получает `WHISPER_BASE_URL=http://whisper-api:8000`.

В админке (`http://localhost:5173`) есть блок **«Голос → Whisper → Bitrix»**:

1. Выберите шаблон, укажите название источника, при необходимости введите **ID сделки** Bitrix.
2. Загрузите аудиофайл и нажмите **«Запустить цепочку»**.

Backend вызывает `POST /transcribe`, сохраняет транскрипт в заявке и пытается выполнить действие в Bitrix по простым правилам текста:

- **следующий этап / дальше** — `crm.deal.update` на следующую стадию воронки;
- **назад / предыдущий** — предыдущая стадия;
- **на стадию … / executing / в работе** и т.п. — переход на распознанную стадию;
- **создай задачу: …** — `tasks.task.add`.

Номер сделки можно произнести в аудио или задать полем **ID сделки**. Первый запуск контейнера Whisper может занять несколько минут (загрузка модели).

### Email approval flow

- email approval сейчас моделируется как persist-нутый approval/task flow
- backend сохраняет status и лог события
- реальный SMTP-отправитель пока не реализован

## Метрики и Grafana

Backend экспортирует Prometheus-метрики для:

- raw HTTP request count
- business-facing HTTP request count (без `/metrics`, health checks и admin polling)
- jobs created total
- current job count by status
- processing duration histogram
- error count
- backend uptime

Grafana провижинится автоматически и содержит dashboard `TSK Backend Overview` с:

- product API requests total
- jobs created
- failed jobs
- total errors
- product traffic / background noise / errors rate
- jobs by status
- p95 job processing duration

`GET /api/v1/health` также возвращает обе сводки отдельно:

- `productRequestsTotal` - продуктовые API-запросы
- `httpRequestsTotalRaw` - весь HTTP traffic, включая технический фон

Admin dashboard обновляет данные раз в 15 секунд только пока вкладка видима. Эти
polling-запросы помечаются служебным заголовком и исключаются из business-метрик,
но продолжают попадать в raw технические счётчики.

## Локальный запуск backend/admin/infra

### Windows PowerShell

Из корня репозитория:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dev-up.ps1
```

После изменений в `apps/backend-api` или `apps/admin-web` не используйте `-SkipBuild`,
иначе Docker может поднять ранее собранный образ.

Если образы уже были собраны ранее:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dev-up.ps1 -SkipBuild
```

Логи:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dev-logs.ps1
```

Остановка:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dev-down.ps1
```

Остановка и удаление контейнеров:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\dev-down.ps1 -RemoveContainers
```

### macOS / Linux / environments with `make`

```bash
make build
make up
```

Логи:

```bash
make logs
```

Остановка:

```bash
make down
```

### Доступные сервисы

- PostgreSQL: `postgres://tsk:tsk@localhost:5432/tsk`
- Whisper (транскрипция): `http://localhost:8000/` и `POST http://localhost:8000/transcribe`
- backend API root: `http://localhost:8080/`
- backend health JSON: `http://localhost:8080/health`
- backend health JSON (versioned): `http://localhost:8080/api/v1/health`
- backend metrics: `http://localhost:8080/metrics`
- admin web: `http://localhost:5173`
- admin-proxied health JSON: `http://localhost:5173/health`
- admin-proxied API health JSON: `http://localhost:5173/api/v1/health`
- admin-proxied metrics: `http://localhost:5173/metrics`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (`admin` / `admin`)

`apps/admin-web` теперь использует same-origin маршруты `/api`, `/health` и `/metrics`.
В Docker они проксируются через `nginx`, а в локальном `vite dev` - через proxy к
`http://localhost:8080`. Это убирает build-time зависимость от захардкоженного
`VITE_API_BASE_URL=http://localhost:8080` для локального сценария.

## Запуск mobile app отдельно

Mobile запускается честно с хоста, а не через Docker-эмуляцию.

```bash
cd apps/mobile-app
npm install
npm run start
```

По умолчанию mobile использует:

- Android emulator: `http://10.0.2.2:8080`
- прочие среды: `http://localhost:8080`

Если нужен другой backend URL, перед запуском задайте `EXPO_PUBLIC_API_BASE_URL`.

Примеры:

```powershell
$env:EXPO_PUBLIC_API_BASE_URL="http://10.0.2.2:8080"
npm run start
```

```powershell
$env:EXPO_PUBLIC_API_BASE_URL="http://192.168.1.50:8080"
npm run start
```

Для физического устройства укажите LAN IP хоста.

## Минимальный локальный сценарий проверки

1. Поднимите backend/admin/infra через `.\scripts\dev-up.ps1` или `make up`.
2. Убедитесь, что health endpoint отвечает JSON:
   - `http://localhost:8080/health`
   - или `http://localhost:5173/api/v1/health`
3. Откройте `http://localhost:5173` и нажмите `Refresh data`.
4. В админке должны загрузиться `Backend status`, templates, jobs и operational feed без `404`.
5. При необходимости загрузите template или используйте seed templates.
6. Запустите mobile app с хоста (`cd apps/mobile-app && npm install && npm run start`).
7. В mobile выберите template, запишите voice note, добавьте notes и отправьте request.
8. Убедитесь, что в админке появились:
   - новый job
   - source document с voice file
   - processing events
   - после обработки - generated document
9. Проверьте `task commands` / dispatch intents:
   - без `BITRIX_WEBHOOK_URL` должен появиться recorded/pending flow
   - с `BITRIX_WEBHOOK_URL` backend попытается выполнить webhook вызов
10. Откройте Grafana и проверьте dashboard `TSK Backend Overview`.

## Как быстро получить видимые метрики

Сразу после старта Prometheus уже должен видеть target `backend-api`.

1. Откройте `http://localhost:9090/targets` и проверьте, что `backend-api` в состоянии `UP`.
2. Откройте `http://localhost:8080/health` 2-3 раза или нажмите `Refresh data` в админке.
3. Создайте хотя бы один job из admin или отправьте voice request из mobile.
4. Для появления processing-метрик дождитесь, пока job перейдет в `completed` или `failed`.
5. После этого откройте `http://localhost:3000` и dashboard `TSK Backend Overview`.

Полезные Prometheus queries для локальной проверки:

- `up{job="backend-api"}`
- `sum(tsk_business_http_requests_total)`
- `sum(tsk_http_requests_total)`
- `sum(tsk_http_requests_total) - sum(tsk_business_http_requests_total)`
- `sum(tsk_document_jobs_created_total)`
- `sum by (status) (tsk_document_jobs_by_status)`
- `sum(tsk_errors_total)`

## Legacy сервисы

Legacy topology по-прежнему поднимается отдельным profile:

```bash
make up-legacy
make logs-legacy
```

Это не часть нового MVP flow и не должно мешать backend/admin/mobile стеку.
