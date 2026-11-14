# repository-structure

backend/services содержит отдельные Go-модули; backend/contracts — proto и сгенерированный Go; backend/go.work связывает модули. backend/infrastructure — Dockerfile и конфигурации; Compose в backend/deployments/local. frontend/saya-app — единственное место Flutter. docs — общая документация; .agents/skills — правила задач. Новый сервис получает собственную БД, миграции и go.mod; не добавлять пустые сервисы.

В каждом backend/services/<service>/ находятся .env.example и локальный игнорируемый .env;
internal/bootstrap/config.go задаёт типизированный контракт конфигурации.
backend/packages/config содержит общий dotenv reader и команду dev-env.
backend/deployments/local/.env остаётся источником настроек инфраструктуры и первоначальной генерации;
Compose использует отдельные service env_file.
