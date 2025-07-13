FROM golang:1.24  as builder
WORKDIR /app
COPY . ./

RUN go mod download
RUN go mod verify

RUN GOOS=linux GOARCH=amd64 go build -tags 'fts5,osusergo,netgo,static' --ldflags '-linkmode external -extldflags "-static"' -o /app/rinha ./cmd/rinha

FROM alpine:latest
RUN apk update && apk add --no-cache libc6-compat

EXPOSE 9999

COPY --from=builder /app/rinha ./rinha
COPY --from=builder /app/.env ./.env
COPY --from=builder /app/service.sh ./service.sh

# Run on container startup.
CMD ["./start.sh"]
