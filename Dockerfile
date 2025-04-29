#
# Start from golang base image
#
FROM golang:1.24-bullseye AS builder

# Set the current working directory inside the container
WORKDIR /build

# Copy go.mod, go.sum files and download deps
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copy sources to the working directory and build
COPY . .
RUN echo "Building app" && make build

#
# Start a new stage from alpine
#
FROM alpine:3.21
LABEL org.opencontainers.image.source=https://github.com/adampresley/sftpslurper

# Create a non-root user and group with specific UID/GID 1000
RUN addgroup -g 1000 -S appgroup && adduser -u 1000 -S appuser -G appgroup

WORKDIR /dist
RUN mkdir -p /dist/uploads && \
   chown -R appuser:appgroup /dist

RUN apk --no-cache add ca-certificates

# Copy the build artifacts from the previous stage
COPY --from=builder /build/cmd/sftpslurper/sftpslurper .

# Set ownership of the application binary
RUN chown appuser:appgroup /dist/sftpslurper

# Switch to non-root user
USER appuser

# Run the executable
ENTRYPOINT ["./sftpslurper"]
