FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/musicclubbot ./cmd/server

FROM gcr.io/distroless/base-debian12
COPY --from=builder /bin/musicclubbot /bin/musicclubbot
EXPOSE 6969
ENTRYPOINT ["/bin/musicclubbot"]
