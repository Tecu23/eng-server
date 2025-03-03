# syntax=docker/dockerfile:1

##########################
# Build Stage
##########################
FROM golang:1.20-alpine AS builder
WORKDIR /app

# Copy go.mod and go.sum to leverage caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code.
COPY . .

# Build the server binary.
# The build command uses -ldflags to inject the version.
# The VERSION argument defaults to "0.1.0" but can be overridden.
ARG VERSION=0.1.0
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X main.version=${VERSION}" -o eng-server ./cmd/server

##########################
# Final Stage
##########################
FROM scratch
# Copy the statically compiled binary from the build stage.
COPY --from=builder /app/eng-server /eng-server

# Expose the port used by the server.
EXPOSE 8080

# Run the binary.
ENTRYPOINT ["/eng-server"]
