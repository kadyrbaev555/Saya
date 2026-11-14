# authorization

Register всегда USER; роль body игнорируется. Subject берётся из проверенного JWT. Survey Input и Assessment Get/Input проверяют совпадение owner в SQL. Auth Realtime допускает только user:{subject}:assessments. ADMIN не специалист. Bootstrap первоначального администратора — отдельный офлайн command, публичного role management endpoint нет.

Локальный gRPC не опубликован на host, но service-to-service authentication/TLS ещё не внедрены; это отдельная production-задача. На legacy локальном PostgreSQL volume Auth bootstrap-user требует отдельно одобренной миграции роли.
