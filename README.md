# SAYA Platform v1.3

Локальная демонстрационная платформа персонализированной поддержки. Реализован сквозной сценарий: добровольный опрос → RabbitMQ → Assessment → Mock LLM → сохранение результата → Centrifugo → Flutter. Это не валидированная оценка риска и не действующий кризисный центр. Используйте демонстрационные данные.

## Запуск

Требования: Go 1.25.1+, Docker Compose, Flutter (проверено 3.50.0 / Dart 3.14.0).

```sh
make dev-init                 # один раз; независимые случайные секреты в игнорируемом .env
make dev-up
make dev-status
make frontend-run            # отдельный процесс, Chrome localhost:3000
make dev-down                # без удаления volumes
```

Gateway http://localhost:8080; WebSocket ws://localhost:8000/connection/websocket; Prometheus http://localhost:9090; Grafana http://localhost:3001 (admin, пароль из локального .env); RabbitMQ management http://localhost:15672. Административные порты привязаны к loopback. Не публикуйте .env и не используйте локальные настройки как production.

В этой среде Flutter установлен только во `/tmp/saya-flutter-sdk`, поэтому используйте `make FLUTTER=/tmp/saya-flutter-sdk/bin/flutter frontend-run`. Некорректный системный GOROOT обходится `GOROOT=/usr/local/go make backend-test`; системные настройки не изменялись.

## Проверки

```sh
make backend-test
make backend-vet
make backend-build
make backend-integration
make api-smoke
make frontend-test
make frontend-integration      # требует работающий Compose и доступный Docker CLI
make frontend-build
make demo-event USER_ID=<uuid> # нейтральное событие для уже подключённого пользователя
```

`backend-integration` проверяет реальную БД Auth и RabbitMQ retry/DLQ; дополнительные Assessment integration tests описаны в docs/development/testing.md. External LLM adapter протестирован локальным TLS mock; реальный внешний провайдер не вызывался. Android/macOS runners созданы, но native builds не проверены: отсутствуют Android SDK, полный Xcode и CocoaPods.

## Структура

- `backend/services`: независимые модули Gateway, Auth, Survey, Assessment, AI.
- `backend/contracts`: Protobuf/generated Go, OpenAPI и JSON event schemas; `backend/go.work` связывает модули.
- `backend/packages`: технические messaging/runtime/outbox adapters.
- `backend/infrastructure`: Docker/Centrifugo/Prometheus/Grafana/RabbitMQ; Compose в `backend/deployments/local/compose.yaml`.
- `frontend/saya-app`: единственный Flutter-проект Web/Android/macOS, Riverpod/go_router/Material 3.
- `docs`, `AGENTS.md`, `.agents/skills`: архитектура, ADR, безопасность и правила продолжения.

Support Service задокументирован как будущий каталог, код пустого сервиса не создаётся. Нет камеры, биометрии, диагнозов, произвольного risk score, автоматических экстренных действий или передачи данных третьим лицам без согласия.

[Инженерный отчёт](docs/development/completion-report.md) · [Фактический статус](docs/development/current-status.md) · [Установка](docs/development/setup.md) · [Архитектура](docs/architecture/overview.md) · [REST](docs/api/rest.md) · [LLM](docs/api/llm.md)

**Локальные данные:** ежедневный Compose использует чистый проект `saya-v13` с разделёнными ролями и новыми секретами. Старый `saya` остановлен; его volumes сохранены, конфигурация находится в игнорируемом `.local/legacy-saya/local.env` (0600). Миграция старой bootstrap-роли была отклонена автопроверкой и не выполнялась; она не нужна для запуска нового окружения.

### Env по сервисам
Каждый сервер использует `backend/services/<service>/.env` и `internal/bootstrap/config.go`.
Для существующей установки выполните `make service-env`, затем `make dev-up`.
Новая установка: `make dev-init` создаёт все файлы автоматически.
Настройки Gemini находятся в `backend/services/ai-service/.env`.
Приоритет окружения, ENV_FILE и правила ротации описаны в [setup](docs/development/setup.md).
