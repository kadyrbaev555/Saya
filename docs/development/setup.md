# setup

Требуются Go 1.25.1+, Docker Compose, Flutter 3.50.0/Dart 3.14 (проверенная версия). Здесь Flutter установлен во /tmp/saya-flutter-sdk, системная установка не менялась.

Из корня: make dev-init (только один раз) → make dev-up → make dev-status. Остановка make dev-down не удаляет volumes. make dev-logs показывает логи. Запуск UI: make frontend-run; порт 3000 включён в Origin allowlist. Gateway localhost:8080, Centrifugo localhost:8000, Prometheus localhost:9090, Grafana localhost:3001 (admin, пароль GRAFANA_PASSWORD из локального .env), Rabbit management localhost:15672 (saya, RABBIT_PASSWORD). Не выводить .env в общие логи.

Если GOROOT указывает на другую версию, GOROOT=/usr/local/go make backend-test. Временный Flutter: make FLUTTER=/tmp/saya-flutter-sdk/bin/flutter frontend-test. Android emulator: --dart-define=API_BASE_URL=http://10.0.2.2:8080 --dart-define=WS_URL=ws://10.0.2.2:8000/connection/websocket. Native build требует Android SDK или полный Xcode+CocoaPods. MacOS Keychain требует keychain-access-groups entitlement. HTTPS deployment включает COOKIE_SECURE=true и allowlisted HTTPS Origin.

Существующий старый volume: не запускать ручное удаление или переименование ролей без проверки ownership и одобрения; см. current-status.


Первый ADMIN: передайте JSON {email,password} через stdin в make admin-bootstrap из защищённого локального источника (не аргументы shell и не историю команд). Команда отказывает при существующем ADMIN. Проверена на одноразовой БД. По умолчанию администратор с предсказуемым паролем не создаётся.

Текущий проект Compose: saya-v13, чистые отдельные volumes. Старый проект saya остановлен, его volumes сохранены, конфигурация сохранена в игнорируемом .local/legacy-saya/local.env (0600). Не используйте её для нового проекта. Для старых данных есть только review-only SQL, автоматическая миграция не запускалась.

## Конфигурация каждого сервиса

Пять серверов читают типизированный Config через собственный
`backend/services/<service>/internal/bootstrap/config.go`.
Файлы `.env` и `.env.example` находятся в каталоге каждого сервиса:
auth-service, api-gateway, survey-service, assessment-service, ai-service.

- Новый checkout: `make dev-init` создаёт инфраструктурные секреты и сервисные файлы.
- Существующая установка: `make service-env` создаёт только отсутствующие сервисные файлы из существующего `backend/deployments/local/.env`; существующие файлы не перезаписываются.
- Изменения применяются через `make dev-up` (пересоздание контейнеров; простой restart не перечитывает env_file).
- Gemini и другие LLM-настройки меняйте в `backend/services/ai-service/.env`.
- Compose передаёт каждому серверу только его env_file. Миграции читают DATABASE_URL того же сервиса.
- Инфраструктурный файл используется для provisioning БД, RabbitMQ, Centrifugo, Grafana и вспомогательного demo-инструмента. Сохранённые в нём старые сервисные настройки — источник первоначального переноса, а не активная конфигурация серверов.

Загрузка: окружение процесса > выбранный dotenv-файл > значения по умолчанию.
Даже явно пустая переменная окружения перекрывает файл; обязательная пустая настройка — ошибка.
По умолчанию читается `.env` текущего каталога. Его отсутствие допустимо, если настройки переданы окружением.
`ENV_FILE` задаёт явный путь: отсутствие/ошибка такого файла останавливает запуск.
Парсер godotenv v1.5.1 поддерживает кавычки и комментарии; загрузчик не меняет глобальное окружение.
Не печатайте Config или разрешённый Compose config: они содержат секреты.

Пример запуска из каталога backend:

```sh
ENV_FILE=services/auth-service/.env go run ./services/auth-service/cmd/server
```

Сгенерированные адреса относятся к сети Docker. Для запуска Go на хосте задайте доступные адреса зависимостей:
PostgreSQL доступен с хоста по адресу 127.0.0.1:5434; внутри Docker используется postgres:5432.
RabbitMQ по умолчанию не публикуется на хост.
Параметры Argon2 и COOKIE_SECURE проверяются типизированно. COOKIE_SECURE по умолчанию true;
локальный gateway .env явно задаёт false для HTTP localhost.
HTTP LLM требует HTTPS endpoint и ключ; Gemini требует ключ и модель; mock работает без ключа.

Секреты создаются с правами 0600 и исключены из Git/Docker build context.
Изменение паролей инфраструктуры само по себе не меняет существующие БД/volumes:
согласуйте ротацию с сервисными DATABASE_URL/RABBIT_URL и миграциями.
Подписывающий JWT_SEED и проверяющий JWT_PUBLIC/JWT_PUBLIC_KEYS также должны соответствовать друг другу.

Native-запуск: [macOS и Android](native-testing.md). Команды: make frontend-macos, make frontend-android (FLUTTER и ANDROID_DEVICE настраиваются).
