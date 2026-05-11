# Architecture Notes

## Monorepo boundaries

- `apps/backend-api` owns the new HTTP API and business flows
- `apps/admin-web` owns operator-facing admin workflows
- `apps/mobile-app` owns field/mobile workflows
- `infra` owns local orchestration and observability
- `bot-service` and `transc-python-service` remain legacy POC services

## Backend layering

The backend is intentionally small but structured:

- `domain` contains document template/job entities and repository interfaces
- `application` contains use cases such as creating document jobs
- `interfaces/http` maps HTTP requests to application services
- `infrastructure` provides config, metrics, and in-memory adapters

## Frontend direction

The admin and mobile apps use the same high-level separation:

- `app` for composition and bootstrapping
- `pages` or `screens` for route-level UI
- `entities` for domain-facing types
- `features` for user actions and data flows
- `shared` for API access and cross-cutting helpers

## Next incremental steps

1. Replace in-memory repositories with a database-backed adapter.
2. Add authentication and authorization to the admin/web API path.
3. Introduce a shared API contract package if the frontend/mobile surfaces grow.
4. Fold useful logic from the legacy bot/transcription POC into the new backend.
