# Development Dockerfile for Go Microservices
FROM golang:1.21-alpine

WORKDIR /app

# Install git for downloading go modules
RUN apk add --no-cache git

# Copy the go.mod and go.sum files (we only have go.mod for now)
COPY go.mod ./
# COPY go.sum ./

# Download all dependencies.
RUN go mod download

# Copy the source code
COPY . .

# We don't specify an ENTRYPOINT/CMD here because docker-compose will
# override it based on which microservice it wants to run.
