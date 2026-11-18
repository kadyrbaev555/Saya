# rest

Контракт backend/contracts/openapi/auth.json дополняется workflow.json. Auth register/confirm и login возвращают access_token,user; браузер получает HttpOnly refresh cookie и csrf_token. GET /auth/csrf восстанавливает CSRF token после перезагрузки; POST refresh/logout требует matching header. Native клиент без Origin отправляет X-Saya-Client:native и получает opaque refresh_token в response для OS secure storage; в refresh/logout передаёт его JSON. Нельзя хранить его в preferences/localStorage.

Все user endpoints защищены Bearer JWT. GET surveys/current — версия demo; POST surveys/submissions → 202 {id,status}; GET assessments/{id} — pending/completed/failed и детерминированные рекомендации. User ID берётся только из JWT, а ownership проверяет сервис. Realtime POST connection-token, subscription-token с {channel}; чужой канал отклоняется. /.well-known/jwks.json — несколько активных/retiring Ed25519 public keys. /health/live, /health/ready, /metrics — операционные endpoints.

## Регистрация с подтверждением email

Первый POST /api/v1/auth/register принимает full_name, phone (с кодом страны), email и password. После приёма письма SMTP-сервером API возвращает 202 и registration_id. Код из шести цифр действует 10 минут. При повторной отправке через минуту используйте новый registration_id. После пяти неверных попыток потребуется новый код.

POST /api/v1/auth/register/confirm принимает registration_id и code как строки. Успешный ответ 201 создаёт обычную сессию Web или native. Неверный, использованный или просроченный код возвращает 401.

## Опрос о неделе

GET /api/v1/surveys/interview возвращает версию 2 и шесть вопросов с утверждёнными вариантами формулировок. POST /api/v1/surveys/interview/focus принимает topic, ratings и отдельное согласие consent. GET /api/v1/surveys/current по-прежнему обслуживает клиентов версии 1.

POST /api/v1/surveys/submissions принимает saya-week, version=2, свободные responses, подтверждённые ratings (0–4), summary, выбранные needs и question_focus. Пропущенные оценки отсутствуют в карте. GET /api/v1/surveys/latest возвращает id последнего прохождения владельца или пустую строку. По этому id GET /api/v1/assessments/{id} отдаёт summary, ratings, percentage и directions. При неполной шкале percentage равен null, нулевой итог остаётся числом 0.

## Вакансии в Алматы

POST /api/v1/support/vacancies требует авторизованной сессии. Клиент передаёт подтверждённые поисковые слова в text и код города 160 в area. Gateway обращается к https://api.hh.ru/vacancies и возвращает до десяти записей в исходном порядке HeadHunter. Ответ содержит items с полями title, employer и url. Пустой массив означает отсутствие результатов. При ошибке провайдера клиент открывает поиск на hh.kz по тем же параметрам. Для доступа зарегистрированного приложения настройте HH_APP_TOKEN в окружении api-gateway и перезапустите Gateway.

## Weekly follow-up

Authenticated clients read `GET /api/v1/weekly/history`. Use the returned `next_cursor` as `before` to fetch older results. Timestamps are Unix seconds. A zero `next_due` means no follow-up is scheduled. To change Android push consent, send `installation_id` and boolean `enabled` to `POST /api/v1/weekly/device`.
