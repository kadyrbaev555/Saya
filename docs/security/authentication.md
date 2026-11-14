# authentication

JWT EdDSA, 15 минут, issuer saya-auth/audience saya-api, обязательные exp/algorithm/kid. JWT_KID выбирает signing key; JWT_PUBLIC_KEYS JSON map публикует active+retiring keys и проверяется Gateway. Ротация: сначала добавить новый public key, затем сменить Auth seed/kid, старый убрать после 15 минут + clock skew. Это конфигурационная ротация с restart, не удалённый JWKS cache.

Refresh 256-bit opaque, SHA-256 в БД; session 30 дней. FOR UPDATE обеспечивает один победивший refresh, повтор отзывает family. Клиент сериализует refresh. Logout отзывает refresh family; access живёт до TTL. Argon2id PHC с хранимыми параметрами: memory 64–256MiB, iterations 1–6, threads 1–8, defaults 64MiB/3/2; legacy hashes совместимы. Per-IP limiter process-local 120/min, не распределённая защита.

Web: HttpOnly/SameSite Strict/CSRF/Origin allowlist, Secure под HTTPS; native refresh только в secure storage. Bootstrap initial ADMIN: cmd/bootstrap-admin принимает JSON email/password через stdin, блокирует advisory lock, отказывает если ADMIN уже есть; не повышает существующего USER.
