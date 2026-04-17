## Build stage
FROM golang:1.26.2-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates && update-ca-certificates

COPY go.mod ./
RUN go mod download

COPY . ./

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build \
		-ldflags "-s -w \
			-X 'main.Version=${VERSION}' \
			-X 'main.Commit=${COMMIT}' \
			-X 'main.BuildDate=${BUILD_DATE}'" \
		-o /bot ./cmd/bot

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build \
		-ldflags "-s -w" \
		-o /ingest ./cmd/ingest

## Runtime stage
FROM alpine:3.23

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && update-ca-certificates

COPY --from=builder /bot /app/bot
COPY --from=builder /ingest /app/ingest

# TOKEN и USERNAME должны приходить через окружение или --env-file
ENV TOKEN=""
ENV USERNAME=""
ENV TZ="Europe/Moscow"

CMD ["/app/bot"]

