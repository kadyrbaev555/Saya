# grpc

backend/contracts/proto/auth/v1/auth.proto — Auth StartRegistration/ConfirmRegistration/Login/Refresh/Logout/Me/Realtime. backend/contracts/proto/workflow/v1/workflow.proto — Survey Current/Interview/Latest/Submit/Input, Assessment Get/Input/Complete. AI выполняет внутренний InterviewAssistant.Focus для подбора вопроса и использует Assessment при обработке очереди. Пользовательские запросы проходят через Gateway.

Генерация protoc 32.1 + protoc-gen-go 1.36.10 + protoc-gen-go-grpc 1.5.1, module=saya/contracts. Внутренние Input/Complete не публикуются Gateway. Локальная сеть доверенная; production требует TLS и service credentials. USER не может задать user_id через внешние body.

Assessment.History returns completed assessments for the authenticated owner and the weekly deadline. Assessment.SetDevice registers or removes an Android installation for that owner. Gateway supplies the owner from the authenticated session.
