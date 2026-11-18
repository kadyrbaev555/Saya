# Фактическое состояние — 2026-11-14

## Итог реализации

Основные локальные сценарии P0 и P1 реализованы и фактически проверены. Полное закрытие всех пунктов мастер-промпта **не заявлено**: старый dev volume сохранён отдельно без миграции; native builds и реальный внешний LLM ограничены средой/ключами.

### Реализовано

- Структура backend/frontend, пять сервисных Go-модулей, contracts и workspace; domain/application/adapters, shared technical packages.
- Auth: registration USER, login, Argon2id configurable PHC, Ed25519 access JWT 15m, hash-only opaque refresh, atomic rotation/replay family revocation, logout, profile, CSRF/cookies/native transport, initial ADMIN command via stdin, multi-key JWKS/config rotation.
- Gateway: Gin → gRPC, issuer/audience/algorithm/expiry/kid checks, CORS allowlist, per-IP bounded limiter, request IDs, body/time limits, uniform error envelope, health endpoints, Prometheus request/error/latency and dependency availability metrics.
- Survey: versioned demonstration, explicit processing/external consent, validation, ownership, idempotent submit 202, PostgreSQL transaction+outbox.
- Assessment: typed Survey Signal/InputSource port and provenance; explicit threat rule independent of LLM; pending/completed/failed, ownership, idempotent completion, result+notification transaction.
- AI: Rabbit consumer, LLMProvider, marked deterministic mock, real HTTPS structured endpoint adapter, bounded retry/timeout, response schema/enum/size validation, no external call without consent.
- Rabbit durable exchanges/queues, manual ack/prefetch, publisher confirms, bounded retry and DLQ; persistent demo inbox; notification outbox and REST source of truth.
- Centrifugo Redis engine; expiring connection/subscription JWT, exact owned channel, neutral events. Flutter SDK handles reconnect/token renewal and REST polling handles lost events.
- Flutter Web/Android/macOS project: Riverpod, stable go_router guards, Material 3, welcome/register/login/profile/survey/pending/result; secure storage adapters, refresh singleflight, logout, explicit session restore, timeouts/cancellation, disposal.
- Compose: PostgreSQL/RabbitMQ/Redis/Centrifugo/Prometheus/Grafana/Gateway/Auth/Survey/Assessment/AI; migrations/db-init/tool profiles, business healthchecks, persistent volumes, random secrets outside Git. Support boundary documented only.

### Фактически проверено

- Go tests всех workspace modules и go vet; backend Docker builds.
- Auth HTTP smoke: invalid input/password, duplicate, USER role despite role body, login, protected profile, CSRF, refresh rotation, logout. Replay HTTP test также прошёл: consumed token и его successor после replay отклонены.
- PostgreSQL concurrent refresh test: один success, один rejection, family revoked.
- Rabbit integration: publisher confirms/manual acknowledgements, три попытки, DLQ. PASS.
- Assessment PostgreSQL integration в отдельной временной schema: pending, failed с сохранением детерминированного правила, ownership, idempotent Create/Complete, одна notification. PASS.
- AI local TLS tests: valid/invalid/trailing/extra fields/unknown enums, no consent, timeout; application test no-consent zero calls и max3 retries. PASS.
- P1 HTTP E2E: consent, persistence, idempotency/conflict, чужой owner 404, RabbitMQ/Mock LLM, completed с explicit threat rule. PASS.
- Flutter analyze без замечаний, widget/provider tests, Web JS build. Chrome headless действительно отобразил welcome screen.
- Flutter realtime integration: настоящее demo и P1 assessment.completed → Riverpod → REST result. Повторно PASS на centrifuge0.21.0 и secure_storage11.2.0.
- Prometheus up=1 для gateway/centrifugo/rabbitmq, Grafana /api/health 200; datasource/dashboard provisioned.
- Чистый изолированный saya-verify stack на новых volumes: Auth и P1 smoke PASS, все 3 service roles rolsuper=false.
- RabbitMQ outage recovery в одноразовом стеке: поставленное задание завершилось после restart broker; внешний mode без consent дал persisted failed и сохранил правило. PASS.
- Flutter SDK reconnect после реального restart Centrifugo, затем P1 event/REST result. PASS.
- Key rotation в одноразовом стеке: новый signer, overlap старого/нового JWT, JWKS двух ключей, отклонение retired key. PASS.
- Основной saya-v13: service roles без SUPERUSER, Auth role не может подключиться к Survey DB. Prometheus 3 scrape targets и 4 gRPC dependency gauges healthy; Grafana datasource health и provisioned dashboard 200.
- Initial ADMIN bootstrap через stdin и запрет повторного bootstrap проверены в одноразовой БД.
- 6 Skills прошли quick_validate.py.

### Ограничения и согласование

1. Текущий ежедневный проект saya-v13 использует новые volumes и независимые секреты. Старый saya остановлен, volumes сохранены, конфигурация .local/legacy-saya/local.env (0600). Старая локальная роль saya_auth — PostgreSQL bootstrap-superuser OID10. Попытка убрать SUPERUSER вернула штатный отказ PostgreSQL. Альтернативный recovery с переименованием/ownership/login был отклонён автоматической проверкой и **не выполнен**. Подготовлен review-only backend/infrastructure/docker/legacy-auth-role-migration.sql; перед выполнением нужны явное согласие и backup. На новых volumes разделение уже проверено.
2. Локальный dev admin secret попал в диагностический вывод неудачной команды. Он не production secret; старый стек остановлен. Новый проект использует другой независимый secret; смена секрета старой БД включена в запрос согласования. Секреты отсутствуют в tracked files и build context.
3. Android SDK, полный Xcode и CocoaPods отсутствуют: native builds не проверены. Flutter 3.50.0/Dart3.14 временно установлен /tmp/saya-flutter-sdk.
4. Реальный внешний LLM не вызывался без ключа/одобренных условий. HTTPS adapter предназначен для документированного structured endpoint, а не произвольного vendor envelope.
5. Production service TLS/authentication, distributed rate limiting, retention/account deletion jobs и организационная кризисная поддержка не реализованы. Это отдельные production/следующие итерации, не гарантии текущего MVP.
6. Logout отзывает refresh; access JWT действует до 15m. Не гарантируется безопасность скомпрометированного устройства/удаление истории браузера.

Системный GOROOT ошибочно указывает 1.26.0 при Go1.25.1; команды использовали GOROOT=/usr/local/go. Текущие данные — только локальные demo/test accounts. Не выполнять push или production deploy без отдельной авторизации.

Основной backend saya-v13 и статическая Web-сборка на localhost:3000 оставлены запущенными. Одноразовый saya-verify остановлен с сохранением volumes. Старый saya также остановлен; старые данные не удалены.

Итоговый runtime check: все 11 контейнеров saya-v13 healthy; /health/ready, Centrifugo /health, Prometheus /-/healthy, Grafana /api/health и Web :3000 возвращают HTTP200. Прямой Centrifugo security test PASS: подписка без JWT отклонена, JWT чужого channel закрывает соединение с invalid-token, повторное разрешённое подключение доставляет нейтральное событие.

## Продолжение

Опционально согласовать миграцию старых dev данных/ролей с backup; для нового стека это не требуется. Проверить native runners на машине с SDK/Xcode; подключать внешний LLM только после условий и consent. Далее P2: retention/deletion, service authentication/TLS, более широкие outage/reconnect tests, доступный каталог Support с проверенными организациями. Архитектура и ADR актуальны в docs/architecture; команды в README/Makefile.


## P2 — Gemini integration (продолжение)

Добавлен отдельный Gemini REST adapter, конфигурация GEMINI_API_KEY/GEMINI_MODEL, проверка provider в Assessment. Ключ/модель ожидаются; внешний Gemini не вызывался и активный mock-режим не переключён. Survey и approved recommendation catalogue по-прежнему демонстрационные. Production готовность не заявляется. Проверки adapter выполняются локальным HTTP transport fake без отправки данных Google; итог тестов указан в ответе сессии. Следующие блоки: live synthetic Gemini E2E → service TLS/auth → privacy lifecycle → backup/restore → native releases/операционные проверки.

## Обновление: сервисные env и bootstrap config

- Для всех пяти Go-серверов добавлены собственные .env.example и internal/bootstrap/config.go; локальные .env созданы из существующих секретов, без их замены.
- Общий saya/config использует godotenv v1.5.1, читает ENV_FILE/локальный .env без изменения process env; process env имеет приоритет.
- Типизированные параметры Argon2/COOKIE_SECURE, проверка обязательных ключей, URL и адресов, условная валидация LLM provider.
- Compose использует отдельные env_file; миграции берут DATABASE_URL соответствующего сервиса.
- make service-env создаёт отсутствующие файлы, не перезаписывая существующие; make dev-init вызывает его автоматически.
- Go tests, go vet, go build и compose config --quiet прошли. Добавлены проверки dotenv/precedence/redaction/typed values и bootstrap validation.
- Сервисные настройки из старого общего .env больше не управляют серверами. Активные LLM-настройки: backend/services/ai-service/.env.

- Docker: образы пяти сервисов собраны, миграции успешно завершились с сервисными DATABASE_URL; все 11 runtime-контейнеров healthy.
- Повторно пройдены HTTP auth smoke (registration/login/profile/CSRF/rotation/logout) и P1 workflow (consent, persistence, idempotency, ownership, RabbitMQ/AI, completed result).
- Проверена повторная генерация service-env: содержимое существующих файлов сохранено, права всех пяти .env — 0600.

## PostgreSQL: порт для подключения с хоста

Compose публикует PostgreSQL на 127.0.0.1:5434 (контейнерный порт 5432). Внутренние DATABASE_URL используют postgres:5432.

## Подготовка native-проверки macOS/Android

- flutter doctor: Flutter 3.50.0 доступен; Xcode отсутствует (только Command Line Tools), CocoaPods отсутствует, Android SDK/emulator не обнаружены.
- Выполнены реальные попытки flutter build macos --debug и flutter build apk --debug: сборки остановлены отсутствием Xcode/Android SDK. Native запуск не выполнен.
- Добавлены Makefile frontend-macos/frontend-android с адресами эмулятора; Android INTERNET permission перенесён также в основной manifest, debug network config разрешает HTTP только к localhost/127.0.0.1/10.0.2.2.
- Android XML синтаксически проверен. Подробная установка и ручной тест-план: docs/development/native-testing.md.

## Android emulator доступен

Обнаружен emulator-5554, Android 17 API37 ARM64; Android SDK37 и JDK25 из Android Studio, лицензии приняты, Android toolchain в flutter doctor — PASS. Начат первый native debug build. Xcode-проверка отложена пользователем. Backend: все 11 runtime-контейнеров healthy.

### Android native build и UI — фактическая проверка

Debug APK успешно собран (первая сборка 320.6s) и установлен на Pixel_10_Pro/API37 ARM64. Установлены необходимые NDK r28c, SDK35/36 и CMake. После установки эмулятор потерял ADB (offline); перезапуск ADB и холодная загрузка без удаления данных восстановили соединение. Через am start запущена SAYA; uiautomator подтвердил Welcome и форму регистрации. Полный auth/survey/realtime native E2E пока не подтверждён: при вводе данных команды снова задерживаются. API37 принудительно использует software GLES согласно журналу эмулятора. APK: frontend/saya-app/build/app/outputs/flutter-apk/app-debug.apk (не Git).

## 18 ноября 2026 — регистрация с подтверждением email

Flutter запрашивает ФИО, контактный телефон, email и пароль. Auth создаёт временную заявку, отправляет шестизначный код через SMTP и при подтверждении создаёт пользователя и сессию в одной транзакции. Вход ранее созданных пользователей сохранён. Контракты REST и gRPC и сценарии тестовой регистрации обновлены.

Проверены миграции на отдельной PostgreSQL 16.4, истечение срока, лимит попыток, повторная отправка, сбой доставки, конкурирующие подтверждения и вход старого аккаунта. Передача письма проверена локальным SMTP-сервером с STARTTLS. Go test и vet прошли для Auth, Gateway и контрактов. Flutter 3.50.0: analyze без замечаний, четыре теста прошли, сборка Web успешна. Realtime E2E пропущен без запущенного стека. Полный Compose E2E, Android и доставка на настоящий почтовый ящик в этой итерации не проверены.

В этой копии проекта существующий SMTP-компонент не найден. Для реального провайдера предусмотрен SMTP-адаптер в Auth и конфигурация в backend/services/auth-service/.env.example. Следующая проверка — пройти регистрацию с настроенным почтовым сервером до открытия профиля.

## Этап 2 — главная и гостевая навигация

После регистрации, входа и восстановления сессии открывается главная с пятью направлениями. Из неё доступен профиль с опросом. Карточки ведут к карте Krisha, поиску hh.kz по введённым словам, черновику обращения к психологу или волонтёрам и справочнику экстренных номеров Казахстана. Экстренная помощь доступна также с приветственного экрана. Гостевой каталог открывается с приветствия и обеих форм авторизации и показывает пустое состояние. Публикация анкет и платежи относятся к этапу 6.

Проверки Flutter 3.50.0: analyze без замечаний, шесть тестов прошли, Web-сборка успешна. Проверены гостевые переходы и возврат, справочник до входа, главная после регистрации, переход во все пять разделов и путь из профиля к опросу. Realtime E2E пропущен без поднятого стека. Системный набор номера на физическом телефоне не проверялся.
