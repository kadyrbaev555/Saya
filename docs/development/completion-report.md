# Инженерный отчёт SAYA v1.3

## 1. Файлы и директории

Созданы backend/services/{api-gateway,auth-service,survey-service,assessment-service,ai-service}, contracts/{proto,auth,workflow,openapi,events}, packages/{messaging,platform}, infrastructure/{docker,rabbitmq,centrifugo,prometheus,grafana}, deployments/local. Redis engine настроен через Compose и конфиг Centrifugo; отдельный пустой каталог не добавлялся. Frontend единый frontend/saya-app с lib/{app,core,features}, web/android/macos runners, pubspec.lock и test. В docs добавлены product/architecture/security/development/api, семь ADR. Корень содержит AGENTS.md, Makefile, README, git/docker ignore; шесть .agents/skills проверены validator.

## 2. Разделение

Весь Go внутри backend, Flutter внутри frontend/saya-app. Отдельный go.mod каждого процесса, backend/go.work и replace для локальных контрактов/пакетов. Frontend не импортирует backend, не подключается к SQL/Rabbit/Redis/gRPC. Общие версии REST/Protobuf/events задокументированы.

## 3. Реализованные сервисы

Auth — Argon2id/config/JWT/multi-key JWKS/refresh/logout/profile/initial ADMIN/realtime authorization. Gateway — HTTP/gRPC/middleware/metrics. Survey — immutable demo version/consent/ownership/idempotency/outbox. Assessment — typed signals/rules/job statuses/result/notification outbox. AI — stateless Rabbit worker/LLMProvider/mock/HTTPS adapter/validation/bounded retries. Hexagonal boundaries отделяют domain/application и infrastructure adapters.

## 4. Планируемые границы

Support Service только описан как будущий каталог, без фиктивного кода и обещания кризисного центра. Production authentication/TLS между сервисами, retention/deletion и организационная работа специалистов остаются следующими эксплуатационными задачами. Камера/биометрия не реализованы.

## 5. Compose

Запущен отдельный основной проект saya-v13: PostgreSQL, RabbitMQ, Redis, Centrifugo Redis engine, Prometheus, Grafana, Gateway, Auth, Survey, Assessment, AI. Db-init и три Goose migration job завершились. Бизнес-сервисы имеют healthchecks и restart policy. HTTP/admin ports loopback, DB/broker service ports не опубликованы.

Старый saya остановлен, volumes сохранены; .local/legacy-saya/local.env — защищённый снимок прежней конфигурации. Одноразовый saya-verify проверял fresh install/key rotation/outage и остановлен. Старые данные/роли не менялись recovery-скриптом.

## 6. API

POST auth/register, login, refresh, logout; GET auth/csrf, users/me; GET surveys/current, POST surveys/submissions (202); GET assessments/{id}; POST realtime connection-token/subscription-token; JWKS, live/ready/metrics. Контракты OpenAPI auth.json/workflow.json и Proto saya.auth.v1/saya.workflow.v1. Нативный и cookie transport раздельны, owner из JWT.

## 7. Тесты

Go tests/vet/build всех модулей. Реальный PostgreSQL concurrent refresh/replay; Assessment pending/failed/ownership/idempotency/notification transaction. Rabbit confirmed publish/manual ack/retry3/DLQ. LLM TLS tests schema/enum/consent/timeout; application max3 attempts. Auth HTTP invalid data/password/duplicate/role escalation attempt/CSRF/refresh replay/logout. Initial ADMIN bootstrap и duplicate prevention. JWT expiry/wrong signature и multi-key rollout/retirement.

Flutter analyze без замечаний; auth/provider and widget flow login→consent→survey→result; реальный Rabbit/Centrifugo/Riverpod/REST и reconnect после restart Centrifugo. Web build успешен, headless Chrome отобразил UI. Android/macOS builds не запускались из-за недостающих SDK/Xcode/CocoaPods.

## 8. Интеграции

HTTP→gRPC→PostgreSQL, Survey→outbox→Rabbit→Assessment→outbox→AI mock→Completion→notification→Centrifugo→Flutter проверены. Отдельно failed без external consent и broker restart recovery. Fresh service roles rolsuper=false; запрет Auth→Survey DB подтверждён. Prometheus scrape up и 4 dependency gauges healthy; Grafana datasource health/dashboard подтверждены HTTP200.

## 9. Открытые ограничения

Реального внешнего LLM вызова нет без ключа/проверенных условий; HTTP adapter реализует documented structured endpoint, не любой vendor envelope. Native сборки ограничены окружением. Production service TLS/auth, retention/deletion и distributed rate limit пока не заявлены выполненными. Access JWT остаётся valid до15m после logout. Скомпрометированное устройство не защищено гарантиями приложения.

Автопроверка отклонила legacy role recovery как рискованное изменение ownership/credentials. Этот вариант не выполнялся; безопасная альтернатива — новый независимый проект с обычными ролями и новыми секретами. Старый dev admin secret попал в diagnostic exception, поэтому старый стек остановлен, secret не переиспользован. Review-only SQL сохранён исключительно для отдельно одобренной работы со старыми dev данными.

## 10. Команды

make dev-init; make dev-up; make dev-status; make frontend-run; make api-smoke; make backend-test backend-vet backend-build backend-integration assessment-integration; make frontend-test frontend-integration frontend-build; make demo-event USER_ID=<uuid>; make dev-down. Admin bootstrap получает protected JSON по stdin: make admin-bootstrap. В этой среде GOROOT=/usr/local/go и FLUTTER=/tmp/saya-flutter-sdk/bin/flutter передаются для соответствующих команд.

## 11. Решения

ADR001–007 описывают microservices/hexagonal, Gin/sqlx, Rabbit/outbox/Centrifugo/Redis, Flutter/Riverpod/monorepo, LLM port, strict refresh replay и typed workflow. Assessment id совпадает с submission operation id; duplicate submit не создаёт новый результат. LLM выдаёт approved enums, deterministic rule не зависит от модели. PostgreSQL — источник истины, WS — уведомление.

## 12. Продолжение

Работать от current-status и существующих контрактов. Для production отдельно закрывать TLS/service credentials, retention/deletion и threat-model gaps. На подготовленной машине проверить Android/macOS runners; внешний provider включать только после условий/consent. Support наполнять только проверенными организациями при реальной организационной готовности. Старую БД при необходимости мигрировать с backup и отдельным согласием; ежедневный saya-v13 от неё не зависит.
