FROM golang:alpine as builder

WORKDIR /src
COPY . .

RUN apk add --no-cache --virtual .build-deps \
		git
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \ 
	go build -a -installsuffix cgo -ldflags="-w -s" -o game-service

FROM scratch

WORKDIR /app
COPY --from=builder /src/game-service .
COPY logger/logger-config.json logger/logger-config.json

VOLUME ["/var/log/dmstudio"]

EXPOSE 8082
CMD ["./game-service"]
