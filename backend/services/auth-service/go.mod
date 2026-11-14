module saya/auth-service

go 1.25.0

require (
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/google/uuid v1.6.0
	github.com/jmoiron/sqlx v1.4.0
	github.com/lib/pq v1.10.9
	golang.org/x/crypto v0.43.0
	google.golang.org/grpc v1.76.0
	saya/contracts v0.0.0
)

require (
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	golang.org/x/net v0.45.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace saya/contracts => ../../contracts

require saya/messaging v0.0.0

replace saya/messaging => ../../packages/messaging

require saya/config v0.0.0

replace saya/config => ../../packages/config
