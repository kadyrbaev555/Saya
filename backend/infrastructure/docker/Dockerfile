FROM golang:1.25.1-alpine AS build
WORKDIR /src/backend
COPY backend/ ./
ARG SERVICE
ARG COMMAND=server
RUN go build -o /probe ./packages/platform/cmd/probe && go build -o /server ./services/${SERVICE}/cmd/${COMMAND}
FROM alpine:3.22.2
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 saya
COPY --from=build /server /server
COPY --from=build /probe /probe
USER saya
ENTRYPOINT ["/server"]
