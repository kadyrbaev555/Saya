# SAYA Platform v1.3

Локальная демонстрационная платформа персонализированной поддержки. Реализован сквозной сценарий: добровольный опрос → RabbitMQ → Assessment → Mock LLM → сохранение результата → Centrifugo → веб-клиент React или Flutter. Это не валидированная оценка риска и не действующий кризисный центр. Используйте демонстрационные данные.

## Запуск

Для веб-версии нужны Go 1.25.1+, Docker Compose, GNU Make и Node.js 22.12+ с npm (проверено на Node.js 24). Команды выполняются из корня проекта.

```sh
npm ci --prefix frontend/web
make dev-init                 # при первой установке
make dev-up
make dev-status
make frontend-run             # открыть http://localhost:3000
```

Для регистрации настройте SMTP по инструкции в [руководстве по тестированию](docs/development/testing.md). Для остановки веб-клиента нажмите Ctrl+C. Команда `make dev-down` останавливает бэкенд с сохранением данных.

Gateway http://localhost:8080; WebSocket ws://localhost:8000/connection/websocket; RabbitMQ management http://localhost:15672. Административные порты привязаны к loopback. Не публикуйте .env и не используйте локальные настройки как production.

## Проверки

```sh
make backend-test
make backend-vet
make backend-build
make backend-integration
make api-smoke
make frontend-test            # TypeScript
make frontend-integration      # Playwright на production-сборке, требования ниже
make frontend-build
make demo-event USER_ID=<uuid> # нейтральное событие для уже подключённого пользователя
```

`backend-integration` проверяет реальную БД Auth и RabbitMQ retry/DLQ; дополнительные Assessment integration tests описаны в docs/development/testing.md. External LLM adapter протестирован локальным TLS mock; реальный внешний провайдер не вызывался. Android/macOS runners созданы, но native builds не проверены: отсутствуют Android SDK, полный Xcode и CocoaPods.

## Структура

- `backend/services`: независимые модули Gateway, Auth, Survey, Assessment, AI.
- `backend/contracts`: Protobuf/generated Go, OpenAPI и JSON event schemas; `backend/go.work` связывает модули.
- `backend/packages`: технические messaging/runtime/outbox adapters.
- `backend/infrastructure`: Docker/Centrifugo/RabbitMQ; Compose в `backend/deployments/local/compose.yaml`.
- `frontend/web` — основной веб-клиент на React, TypeScript и Vite.
- `frontend/saya-app` — Flutter-клиент для мобильной и настольной разработки.
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

## Веб-клиент

Сборка `make frontend-build` создаёт статические файлы в `frontend/web/dist/`. Для размещения настройте раздачу этих файлов, fallback на `index.html` для клиентских маршрутов, проксирование `/api/` в Gateway и WebSocket `/connection/websocket` в Centrifugo с передачей Upgrade. При отдельном домене Centrifugo задайте `VITE_WS_URL` на этапе сборки. Для локального запуска Vite уже настроен.

Браузерные тесты запускаются командой `make frontend-integration`. Они собирают сайт и поднимают preview на свободном порте. Нужны работающие Go-сервисы и тестовый SMTP-приёмник Mailpit. Перед первым запуском выполните `npx playwright install chromium` в `frontend/web/`. Для полного прогона задайте учётные данные тестового модератора. Настройки стенда и браузера описаны в [инструкции по тестированию](docs/development/testing.md).

Flutter доступен в `frontend/saya-app/`. Команды `make frontend-android`, `make frontend-macos`, `make flutter-test` и `make flutter-integration` используют Flutter SDK.
