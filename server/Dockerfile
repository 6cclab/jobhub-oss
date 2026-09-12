# syntax=docker/dockerfile:1

### Build stage ###
FROM golang:1.26-alpine AS build

WORKDIR /src

# Cache go mod downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/jobhub ./cmd/server/main.go

### Final stage ###
FROM alpine:3.21

# alpine (not distroless) because we may shell out to weasyprint later
RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build /out/jobhub /app/jobhub

ENV PORT=8080

EXPOSE 8080

ENTRYPOINT ["/app/jobhub"]
