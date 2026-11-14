# grpc

backend/contracts/proto/auth/v1/auth.proto — Auth Register/Login/Refresh/Logout/Me/Realtime. backend/contracts/proto/workflow/v1/workflow.proto — Survey Current/Submit/Input, Assessment Get/Input/Complete. AI не предоставляет пользовательский REST/gRPC: consumer использует Assessment.

Генерация protoc 32.1 + protoc-gen-go 1.36.10 + protoc-gen-go-grpc 1.5.1, module=saya/contracts. Внутренние Input/Complete не публикуются Gateway. Локальная сеть доверенная; production требует TLS и service credentials. USER не может задать user_id через внешние body.
