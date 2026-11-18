# rest

Контракт backend/contracts/openapi/auth.json дополняется workflow.json. Auth register/confirm и login возвращают access_token,user; браузер получает HttpOnly refresh cookie и csrf_token. GET /auth/csrf восстанавливает CSRF token после перезагрузки; POST refresh/logout требует matching header. Native клиент без Origin отправляет X-Saya-Client:native и получает opaque refresh_token в response для OS secure storage; в refresh/logout передаёт его JSON. Нельзя хранить его в preferences/localStorage.

Все user endpoints защищены Bearer JWT. GET surveys/current — версия demo; POST surveys/submissions → 202 {id,status}; GET assessments/{id} — pending/completed/failed и детерминированные рекомендации. User ID берётся только из JWT, а ownership проверяет сервис. Realtime POST connection-token, subscription-token с {channel}; чужой канал отклоняется. /.well-known/jwks.json — несколько активных/retiring Ed25519 public keys. /health/live, /health/ready, /metrics — операционные endpoints.

## Регистрация с подтверждением email

Первый POST /api/v1/auth/register принимает full_name, phone (с кодом страны), email и password. После приёма письма SMTP-сервером API возвращает 202 и registration_id. Код из шести цифр действует 10 минут. При повторной отправке через минуту используйте новый registration_id. После пяти неверных попыток потребуется новый код.

POST /api/v1/auth/register/confirm принимает registration_id и code как строки. Успешный ответ 201 создаёт обычную сессию Web или native. Неверный, использованный или просроченный код возвращает 401.
