# events

Rabbit envelope version=1: event_id,user_id,operation_id. Только технические UUID, без ответов/текста. Exchange saya.events.v1 direct/durable; saya.dead.v1 — DLX. Очереди assessment/ai/demo v1 durable с DLQ. Ожидаются повторы; Create/Complete идемпотентны, demo_inbox durable. Временное demo CLI хранится в Auth только для P0 проверки и не является новым продуктовым сервисом.

Centrifugo event: {event:assessment.completed|assessment.failed,assessment_id,status,version:1}; demo.completed для smoke. Канал user:{id}:assessments, JWT connection/subscription expiry 5 минут, owner проверяет Auth. Уведомление не содержит анализа. Не публиковать API key клиенту.
