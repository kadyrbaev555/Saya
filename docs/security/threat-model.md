# threat-model

Риски: доступ к устройству, XSS, кража bearer, CSRF, brute force, replay refresh, компрометация внутренней сети. Меры: короткий access TTL, HttpOnly refresh, CSRF, Origin allowlist, ограничение размера запроса и общий rate limit. Общий limiter не заменяет распределённую защиту от brute force. Logout отзывает refresh, уже выданный access живёт до 15 минут. Безопасность скомпрометированного устройства не гарантируется.
