# rest

Контракт backend/contracts/openapi/auth.json дополняется workflow.json. Auth register/login возвращают access_token,user; браузер получает HttpOnly refresh cookie и csrf_token. GET /auth/csrf восстанавливает CSRF token после перезагрузки; POST refresh/logout требует matching header. Native клиент без Origin отправляет X-Saya-Client:native и получает opaque refresh_token в response для OS secure storage; в refresh/logout передаёт его JSON. Нельзя хранить его в preferences/localStorage.

Все user endpoints защищены Bearer JWT. GET surveys/current — версия demo; POST surveys/submissions → 202 {id,status}; GET assessments/{id} — pending/completed/failed и детерминированные рекомендации. User ID берётся только из JWT, а ownership проверяет сервис. Realtime POST connection-token, subscription-token с {channel}; чужой канал отклоняется. /.well-known/jwks.json — несколько активных/retiring Ed25519 public keys. /health/live, /health/ready, /metrics — операционные endpoints.
