# Stage 1: Build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY internal/ ./internal/
COPY ./cmd/goscouter/ ./
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o goscouter .

# Stage 3: Final image
FROM alpine:3.19
WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy backend binary
COPY --from=backend-builder /app/goscouter .

# Copy frontend build
COPY --from=frontend-builder /app/frontend/out ./frontend/out

EXPOSE 8080

CMD ["./goscouter", "run"]
