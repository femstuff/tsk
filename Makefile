.PHONY: help build up down restart logs clean

help:
@echo "Команды для управления:"
@echo "  make build    - Собрать контейнеры"
@echo "  make up       - Запустить контейнеры"
@echo "  make down     - Остановить контейнеры"
@echo "  make restart  - Перезапустить контейнеры"
@echo "  make logs     - Показать логи"
@echo "  make clean    - Удалить всё"

build:
docker-compose build

up:
docker-compose up -d

down:
docker-compose down

restart:
docker-compose restart

logs:
docker-compose logs -f

clean:
docker-compose down -v
