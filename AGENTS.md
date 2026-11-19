# SAYA v1.3
Go код только в backend/, Flutter только в frontend/saya-app/. Основной веб-клиент React/TypeScript/Vite находится в frontend/web/. Сервисы имеют отдельные go.mod; workspace backend/go.work. Контракты backend/contracts/, инфраструктура backend/infrastructure/, Compose backend/deployments/local/compose.yaml.
Согласованный стек: Go/Gin/sqlx/PostgreSQL/Goose/gRPC/RabbitMQ/Redis/Centrifugo, Flutter/Riverpod/go_router. Не добавлять ORM, OpenTelemetry, Kubernetes, NATS или CI/CD.
Domain независим от инфраструктуры. Gateway не содержит бизнес-логику. Сервисы не читают чужие БД. Ownership проверяется бизнес-сервисом.
Не логировать пароли, токены, ответы опросов. Не передавать чувствительные данные LLM без отдельного согласия. Нет камеры, диагнозов, автоматических экстренных действий.
Проверять изменённую функциональность и обновлять docs/development/current-status.md. Не заменять обязательную функцию фиктивной реализацией. Не менять структуру без обоснования. Не выдавать непроверенное за завершённое.
