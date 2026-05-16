# Stage 1: Build React Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:1.25-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/server ./cmd/server

# Stage 3: Final Image
FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates

# Copy artifacts
COPY --from=backend-builder /app/server ./
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
COPY schema.sql ./
COPY migrations/ ./migrations/
# Copy static files if they are still used (though mostly for favicon/legacy)
COPY static/ ./static/

RUN echo "JAEGER DASHBOARD: http://localhost:16686"

# Environment variables
ENV PORT=8080
ENV ENV=production
EXPOSE 8080

CMD ["./server"]
