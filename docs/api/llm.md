# llm

LLM_PROVIDER=mock работает без ключа и маркирует provider=demonstration-mock. LLM_PROVIDER=http требует LLM_ENDPOINT (HTTPS) и LLM_API_KEY. HTTP adapter вызывает структурированный LLM endpoint POST с Authorization Bearer, version=1, demonstration=true, input boolean signals, allowed_recommendations и инструкцией. Endpoint должен реализовать этот контракт (например, операторский LLM inference endpoint); прямые vendor chat-completions envelopes не считаются совместимыми без адаптации.

Ответ строго {"recommendations":["review_support_information","consider_trusted_contact","choose_safe_device"]}: от 1 до 3 уникальных enum значений, никаких дополнительных полей/текста, размер до 16 KiB. Timeout 3s на попытку, 3 bounded попытки. Нет consent → внешний запрос не выполняется, assessment failed; детерминированные правила остаются. После успешного LLM данные переносятся в одобренные статические тексты Assessment.

Вызовы реального внешнего провайдера не проверены без ключа. Перед включением оператор отдельно проверяет условия обработки и consent; эта версия допускает только демонстрационные данные. Никаких диагнозов, risk score и автоматических действий.


## Gemini — P2

Добавлен отдельный Gemini REST adapter: generateContent, x-goog-api-key header, responseMimeType/responseJsonSchema, проверка candidates/STOP/blockReason и строгого JSON результата. Model ID задаётся явно, чтобы не полагаться на доступность/срок жизни конкретной модели. Ключ не передаётся в URL, prompt, Flutter или логи. Для старого опроса сохраняется прежний формат сигналов.

В защищённый backend/services/ai-service/.env добавьте LLM_PROVIDER=gemini, GEMINI_API_KEY=<локальный секрет>, GEMINI_MODEL=<доступная стабильная модель>. Не отправляйте ключ в чат и не коммитьте .env. Затем make dev-up пересоберёт сервисы. Без ключа оставляйте LLM_PROVIDER=mock. Для Gemini timeout 10s на попытку, до3 попыток; Rabbit worker deadline60s. Ни один внешний вызов не выполнялся при реализации.

Перед настоящими чувствительными ответами отдельно проверьте тариф и [условия обработки Gemini](https://ai.google.dev/gemini-api/terms). Unpaid Services имеют ограничение на отправку sensitive/confidential/personal information; billing сам по себе не заменяет consent и проверку условий. Первые живые проверки — только синтетические сценарии и отдельный consent в опросе.

API reference: [generateContent](https://ai.google.dev/api/generate-content), [structured outputs](https://ai.google.dev/gemini-api/docs/structured-output). Новый adapter не делает MVP готовым к production: внутренний TLS/auth, data lifecycle, backups и продуктовая экспертиза остаются отдельными задачами.

## Диалоговый опрос второй версии

Подбор формулировки и дополнительные рекомендации имеют отдельные согласия. При включении подбора Gateway передаёт запрос в InterviewAssistant.Focus. AI проверяет согласие, тему и диапазон оценок до обращения к Gemini. Провайдер выбирает один из трёх ракурсов утверждённого вопроса. Flutter отображает текст из каталога Survey. При недоступности провайдера пользователь продолжает с обычными вопросами.

Для итоговой рекомендации AI может передать настроенному провайдеру только оценки, направления и булевы признаки. Свободный текст, подтверждённое описание, идентификатор пользователя и контакты во внешний запрос не включаются. При отсутствии согласия, настройке mock или отказе внешнего провайдера Assessment сохраняет самостоятельную оценку с provider=self-report. Процент вычисляется в Assessment.
