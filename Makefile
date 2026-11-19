FLUTTER ?= flutter
BACKEND_TARGETS = ./packages/config/... ./contracts/... ./packages/messaging/... ./packages/platform/... ./services/auth-service/... ./services/api-gateway/... ./services/survey-service/... ./services/assessment-service/... ./services/ai-service/...
COMPOSE = docker compose --env-file backend/deployments/local/.env -f backend/deployments/local/compose.yaml
.PHONY: dev-init dev-up dev-down dev-logs backend-test backend-build frontend-test frontend-run

dev-init:
	cd backend && go run ./services/auth-service/cmd/dev-init
	$(MAKE) service-env

dev-up:
	$(COMPOSE) up -d --build

dev-down:
	$(COMPOSE) down

dev-logs:
	$(COMPOSE) logs -f --tail=100

backend-test:
	cd backend && go test $(BACKEND_TARGETS)

backend-build:
	cd backend && go build $(BACKEND_TARGETS)

frontend-test:
	cd frontend/web && npm run check

frontend-run: web-run

.PHONY: web-run
web-run:
	cd frontend/web && npm run dev

.PHONY: demo-event
demo-event:
	DEMO_USER_ID=$(USER_ID) $(COMPOSE) run --rm --build demo

.PHONY: backend-vet backend-integration api-smoke frontend-build frontend-integration dev-status
backend-vet:
	cd backend && go vet $(BACKEND_TARGETS)
backend-integration:
	$(COMPOSE) run --rm --no-deps migrate sh -c 'cd backend && TEST_DATABASE_URL="$$DATABASE_URL" go test -v ./services/auth-service/internal/adapters/outbound/postgres ./packages/messaging'
api-smoke:
	python3 backend/tests/smoke.py
	python3 backend/tests/workflow.py
frontend-build:
	cd frontend/web && npm run build
frontend-integration:
	cd frontend/web && npm test

.PHONY: flutter-test flutter-integration
flutter-test:
	cd frontend/saya-app && $(FLUTTER) test
flutter-integration:
	cd frontend/saya-app && SAYA_REALTIME_TEST=1 $(FLUTTER) test test/realtime_integration_test.dart
dev-status:
	$(COMPOSE) ps

.PHONY: admin-bootstrap assessment-integration
admin-bootstrap:
	$(COMPOSE) run --rm --no-deps -T migrate sh -c 'cd backend && DATABASE_URL="$$DATABASE_URL" go run ./services/auth-service/cmd/bootstrap-admin'
assessment-integration:
	$(COMPOSE) run --rm --no-deps migrate-assessment sh -c 'cd backend && TEST_ASSESSMENT_DATABASE_URL="$$DATABASE_URL" go test -v ./services/assessment-service/internal/adapters'

.PHONY: service-env
service-env:
	cd backend && go run ./packages/config/cmd/dev-env

.PHONY: frontend-macos frontend-android
frontend-macos:
	cd frontend/saya-app && $(FLUTTER) run -d macos
ANDROID_DEVICE ?= emulator-5554
frontend-android:
	cd frontend/saya-app && $(FLUTTER) run -d $(ANDROID_DEVICE) --dart-define=API_BASE_URL=http://10.0.2.2:8080 --dart-define=WS_URL=ws://10.0.2.2:8000/connection/websocket
