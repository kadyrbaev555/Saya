# communication

REST /api/v1, gRPC saya.auth.v1 и saya.workflow.v1. Rabbit durable direct exchange saya.events.v1, queues saya.assessment.v1/saya.ai.v1/saya.demo.v1. Worker: manual ack, prefetch 1, confirmed retry через 1/2 секунды, после 3 ошибок DLQ; transport failure переотдаёт сообщение. Контейнеры перезапускаются Compose. Outbox не удаляется до подтверждения; результат и completion идемпотентны.

At-least-once: сбой между publish и commit может повторить сообщение/уведомление. UI всегда перечитывает PostgreSQL-backed REST; события не служат доказательством результата. Centrifugo channel user:{uuid}:assessments требует subscription JWT, expires 5 минут; клиент SDK обновляет токены и reconnect. REST polling pending 2 секунды обеспечивает восстановление даже без WS.
