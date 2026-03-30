FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o kr-metro-api .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates && \
    addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser
COPY --from=builder /app/kr-metro-api /usr/local/bin/
USER appuser
EXPOSE 8080
CMD ["kr-metro-api"]
