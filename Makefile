.PHONY: help build up down restart logs clean up-mobile logs-mobile up-legacy logs-legacy

DEFAULT_SERVICES := postgres redis backend-api admin-web prometheus grafana

help:
	@echo "Команды для управления:"
	@echo "  make build       - Собрать backend/admin для нового стека"
	@echo "  make up          - Запустить backend/admin/infra без mobile"
	@echo "  make down        - Остановить новый стек"
	@echo "  make restart     - Перезапустить новый стек"
	@echo "  make logs        - Показать логи нового стека"
	@echo "  make up-mobile   - Поднять optional mobile profile"
	@echo "  make logs-mobile - Логи mobile profile"
	@echo "  make up-legacy   - Поднять legacy bot + whisper profile"
	@echo "  make logs-legacy - Логи legacy profile"
	@echo "  make clean       - Удалить контейнеры и volume нового стека"

build:
	docker compose build backend-api admin-web

up:
	docker compose up -d --build --wait $(DEFAULT_SERVICES)

down:
	docker compose down

restart:
	docker compose restart $(DEFAULT_SERVICES)

logs:
	docker compose logs -f $(DEFAULT_SERVICES)

up-mobile:
	docker compose --profile mobile up -d mobile-workspace

logs-mobile:
	docker compose --profile mobile logs -f mobile-workspace

up-legacy:
	docker compose --profile legacy-poc up -d whisper-api telegram-bot

logs-legacy:
	docker compose --profile legacy-poc logs -f whisper-api telegram-bot

clean:
	docker compose down -v --remove-orphans
