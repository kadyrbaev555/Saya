# conventions

Порты принадлежат application. Domain не импортирует инфраструктуру. Context и bounded timeout обязательны. Ошибки API не раскрывают DB/credentials. Не логировать тела запросов. Миграции Goose версионируются и выполняются до старта Auth.
