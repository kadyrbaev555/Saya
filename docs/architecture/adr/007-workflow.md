# Асинхронный workflow, idempotency и сигналы

Контекст: HTTP не ждёт LLM; повторы broker неизбежны. Решение: Survey submission id одновременно operation/assessment id, UUID idempotency key уникальна внутри user. Survey и Assessment сохраняют локальные outbox; assessment completion и notification также атомарны. AI stateless и не имеет собственной БД. В domain Assessment типизированный Signal с source type/version; первый источник только survey. Последствия: eventual consistency, краткий pending до записи Assessment, восстановление через Survey Input и REST polling; повторная доставка не создаёт новый результат.

LLM выдаёт только одобренные recommendation enums; статический текст и явные правила остаются в Assessment. Demo CLI colocated с Auth для P0 проверки; не следует использовать его как продуктовый notification workflow.
