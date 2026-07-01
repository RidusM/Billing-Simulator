FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s" \
    -o /bin/bill-stripe-sim \
    ./cmd/bill-stripe-sim

FROM scratch

COPY --from=builder /bin/bill-stripe-sim /bill-stripe-sim

EXPOSE 8080
ENTRYPOINT ["/bill-stripe-sim"]