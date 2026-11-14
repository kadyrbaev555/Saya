module saya/survey-service

go 1.25.0

require (
	github.com/google/uuid v1.6.0
	github.com/jmoiron/sqlx v1.4.0
	google.golang.org/grpc v1.76.0
	saya/contracts v0.0.0
	saya/messaging v0.0.0
	saya/platform v0.0.0
)

require (
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250804133106-a7a43d27e69b // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace saya/contracts => ../../contracts

replace saya/messaging => ../../packages/messaging

replace saya/platform => ../../packages/platform

require saya/config v0.0.0

replace saya/config => ../../packages/config
