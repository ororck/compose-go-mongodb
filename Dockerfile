FROM golang:1.26 as builder

WORKDIR /app

# Dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 go build -o main .

FROM gcr.io/distroless/static-debian12 as runtime

WORKDIR /app

COPY --from=builder /app/main .

CMD ["/app/main"]
