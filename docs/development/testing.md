# testing

make backend-test, make backend-vet, make backend-build проверяют все workspace modules. make backend-integration выполняет PostgreSQL concurrent rotation/logout и Rabbit retry/DLQ в контейнере с локальными test/demo данными. Без DSN Go integration tests skip — это не PASS интеграции.

make api-smoke проверяет Auth и P1 persistence/ownership/idempotency/async completion. make frontend-test проверяет welcome, auth state и UI flow; make frontend-integration проверяет настоящее RabbitMQ → Centrifugo → Riverpod → REST и P1 assessment.completed. make frontend-build собирает JS Web. Опциональный backend/tests/realtime.py требует websockets==15.0.1 (в этой сессии установлен только /tmp/saya-python).

HTTP LLM tests используют локальный TLS httptest server: consent, schema/enum/extra fields, malformed/trailing output, timeout. Реальный внешний LLM по сети не вызывался. Native Android/macOS builds не проверены из-за отсутствия SDK/Xcode.


make assessment-integration проверяет PostgreSQL pending/failed/idempotency/ownership в временной schema. backend/tests/key_rotation.py и failure_recovery.py намеренно ограничены проектом saya-verify и требуют SAYA_DISPOSABLE_TEST=1; не запускать их против основного/production стека. Проверочный стек уже остановлен.


Centrifugo health.enabled включает /health согласно [официальной конфигурации](https://centrifugal.dev/docs/server/configuration#health). Инфраструктурные wget probes проверяют реальные HTTP endpoints; служебные порты loopback.

### Почта для регистрации

В backend/services/auth-service/.env задайте SMTP_ADDRESS (host:port), SMTP_FROM и, если сервер требует авторизацию, SMTP_USERNAME и SMTP_PASSWORD. На порту 465 используется TLS с начала соединения, на других портах требуется STARTTLS. Сертификат должен быть доверен контейнеру Auth. Без настроенной почты регистрация возвращает ошибку доставки. Для просмотра писем тестовых пользователей настройте Auth на отдельный тестовый SMTP-приёмник Mailpit с TLS. Python smoke/workflow используют его HTTP API по адресу из SAYA_TEST_MAILPIT_URL (по умолчанию http://127.0.0.1:8025). Flutter realtime E2E использует заранее подтверждённый тестовый аккаунт из SAYA_TEST_EMAIL и SAYA_TEST_PASSWORD.

## Weekly reminders on Android

Register the Android app `com.example.saya_app` in Firebase. Place its client configuration at `frontend/saya-app/android/app/google-services.json`. Enable the Firebase Cloud Messaging API and grant the sending service account permission to send messages. Set `FCM_SERVICE_ACCOUNT_B64` in the private assessment-service .env to the base64-encoded service account JSON for the same project. Recreate the assessment service, run its Goose migrations, and rebuild Android. Keep credentials outside version control.

On a device with Google Play services, complete an interview and enable reminders. When the deadline passes with the app closed, verify the neutral notification opens the survey and that completion moves the deadline forward. Database integration tests exercise scheduling and retry independently of Firebase.

## Тестовые пожертвования

Примените миграции assessment-service. Анкеты и очередь модерации работают без платёжного кабинета. На последнем опросе подтвердите потребность в финансовой помощи при низкой оценке расходов. Откройте рекомендации, отредактируйте текст и подтвердите его публикацию. В профиле аккаунта с ролью ADMIN откройте модерацию и одобрите анкету. Гость увидит её через «Помочь человеку». Проверьте жалобу, отзыв и новый опрос.

Для интеграционной демонстрации нужен тестовый мерчант Freedom Pay в Казахстане с включёнными тестовыми платежами, сохранением карты получателя и тестовыми выплатами. В приватном окружении assessment-service задайте FREEDOMPAY_TEST_MERCHANT_ID, FREEDOMPAY_TEST_SECRET, FREEDOMPAY_TEST_PAYOUT_SECRET и FREEDOMPAY_RESULT_URL. Последняя переменная имеет вид https://<публичный-gateway>/api/v1/payments/result. На этом же сервере должны быть доступны /api/v1/payments/card и /api/v1/payments/payout. Пересоздайте assessment-service и Gateway, обновите Flutter.

В «Моя анкета» привяжите тестовую карту на странице Freedom Pay. После уведомления провайдера обновите каталог. Гость выбирает тестовую сумму и платит на стороне провайдера. Заказ хранит токен получателя, подтверждённую сумму и идентификаторы тестовых операций. Каждый подтверждённый заказ запускает одну выплату. После потери ответа сервис запрашивает статус по номеру заказа. Если операция отсутствует у провайдера, требуется ручная сверка перед повторной отправкой. Используйте только тестовые ключи и карты провайдера.

[Тестовые карты](https://docs.freedompay.kz/doc-741636) · [Привязка карты](https://docs.freedompay.kz/12006980e0) · [Выплаты](https://docs.freedompay.kz/11621024e0)
