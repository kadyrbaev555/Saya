# services

Реализованы Gateway (REST/JWT/middleware/Prometheus), Auth (Argon2id/JWT/rotation/CSRF transport), Survey (версионированный demo и согласие/idempotency), Assessment (типизированный Survey Signal, правила, result/notification persistence), AI (Rabbit consumer, mock и HTTPS adapter).

Support Service — только архитектурная граница: будущий каталог организаций, специалистов и ресурсов; собственная БД, публичные карточки и администрирование через Gateway. Нет доступа к приватным ответам. Реальный кризисный центр не заявлен. Новый код сервиса пока не создаётся.
