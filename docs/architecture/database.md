# database

Auth: users/sessions/refresh_tokens/demo_inbox; Survey: submissions/outbox; Assessment: assessments/outbox/notifications. UUID, TIMESTAMPTZ, unique constraints и индексы ownership. Outbox и бизнес-запись — одна транзакция; dispatcher FOR UPDATE SKIP LOCKED. Повторная отправка с той же idempotency key возвращает operation id; другое содержимое с той же key отклоняется.

Свежий Compose bootstrap использует saya_owner отдельно от service users; PUBLIC CONNECT отозван. На старом локальном volume исходная роль saya_auth остаётся bootstrap-superuser PostgreSQL: это открытое ограничение dev окружения, не допустимое production-настройкой. Отклонённый recovery не выполняется без отдельного одобрения. Retention/deletion jobs пока отсутствуют.
