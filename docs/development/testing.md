# testing

make backend-test, make backend-vet, make backend-build проверяют все workspace modules. make backend-integration выполняет PostgreSQL concurrent rotation/logout и Rabbit retry/DLQ в контейнере с локальными test/demo данными. Без DSN Go integration tests skip — это не PASS интеграции.

make api-smoke проверяет Auth и P1 persistence/ownership/idempotency/async completion. make frontend-test проверяет welcome, auth state и UI flow; make frontend-integration проверяет настоящее RabbitMQ → Centrifugo → Riverpod → REST и P1 assessment.completed. make frontend-build собирает JS Web. Опциональный backend/tests/realtime.py требует websockets==15.0.1 (в этой сессии установлен только /tmp/saya-python).

HTTP LLM tests используют локальный TLS httptest server: consent, schema/enum/extra fields, malformed/trailing output, timeout. Реальный внешний LLM по сети не вызывался. Native Android/macOS builds не проверены из-за отсутствия SDK/Xcode.


make assessment-integration проверяет PostgreSQL pending/failed/idempotency/ownership в временной schema. backend/tests/key_rotation.py и failure_recovery.py намеренно ограничены проектом saya-verify и требуют SAYA_DISPOSABLE_TEST=1; не запускать их против основного/production стека. Проверочный стек уже остановлен.


Centrifugo health.enabled включает /health согласно [официальной конфигурации](https://centrifugal.dev/docs/server/configuration#health). Инфраструктурные wget probes проверяют реальные HTTP endpoints; служебные порты loopback.
