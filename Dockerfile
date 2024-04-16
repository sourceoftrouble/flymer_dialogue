FROM golang:alpine AS builder
WORKDIR /app
ADD go.mod .
COPY . .
RUN go build -o bot bot.go

FROM alpine
WORKDIR /app
COPY --from=builder /app/bot /app/bot
CMD ["/app/bot"]
