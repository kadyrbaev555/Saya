# overview

SAYA использует пять бизнес/транспорт процессов: Gateway, Auth, Survey, Assessment, AI. Внешний REST → Gin → Protobuf/gRPC. PostgreSQL принадлежит Auth/Survey/Assessment по отдельным БД и ролям; AI stateless. Domain/application/adapters разделены. Shared packages содержат только техническую инфраструктуру RabbitMQ/runtime/outbox.

Опрос сохраняется вместе с outbox. Dispatcher публикует идентификаторы в saya.assessment.v1 с confirms. Assessment получает Survey Input через gRPC, сохраняет типизированный сигнал с provenance и предварительный детерминированный результат плюс AI outbox. AI читает input через Assessment gRPC, запускает LLMProvider и отправляет Completion. Assessment атомарно сохраняет результат и notification outbox. Centrifugo+Redis доставляет нейтральное событие; клиенты React и Flutter перечитывают REST, polling восстанавливает состояние при разрыве.

Локальная внутренняя сеть не заменяет production TLS/service authentication. На существующем dev volume исходный bootstrap Auth user пока superuser: миграция старой роли требует отдельного согласования; свежая установка создаёт обычные service users.

Конфигурация инфраструктуры и приложений разделена: каждый Go-сервис имеет собственные
.env.example и internal/bootstrap/config.go. Общий технический пакет saya/config
обрабатывает dotenv и примитивы валидации; состав и требования настроек определяет
сам сервис. Конфигурация передаётся адаптерам явно, domain/application не читают env.
