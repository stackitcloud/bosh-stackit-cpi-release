FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /go/src/github.com/stackitcloud/stackit-cpi

# Copy module files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Set build arguments for versioning
ARG VERSION="0.0.1"
ARG GIT_SHA="unknown"
ARG BUILD_DATE="unknown"

# Build the CPI with version information
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
  -ldflags "-X github.com/stackitcloud/stackit-cpi/version.Version=${VERSION} \
  -X github.com/stackitcloud/stackit-cpi/version.GitSHA=${GIT_SHA} \
  -X github.com/stackitcloud/stackit-cpi/version.BuildDate=${BUILD_DATE}" \
  -o /go/bin/stackit-cpi .

# Run tests (can be disabled by setting SKIP_TESTS=true)
ARG SKIP_TESTS=false
RUN if [[ "$SKIP_TESTS" != "true" ]]; then go test -v ./...; fi

# Create a minimal runtime image
FROM alpine:3.18

# Install CA certificates for HTTPS and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create a non-root user and group (vcap is the standard BOSH user)
RUN addgroup -g 1000 -S vcap && \
    adduser -u 1000 -S vcap -G vcap

# Create necessary directories
RUN mkdir -p /var/vcap/jobs/stackit-cpi/bin \
  /var/vcap/jobs/stackit-cpi/config \
  /var/vcap/sys/log/stackit-cpi

# Set proper permissions and ownership
RUN chmod 755 /var/vcap/jobs/stackit-cpi/bin \
  /var/vcap/jobs/stackit-cpi/config \
  /var/vcap/sys/log/stackit-cpi && \
  chown -R vcap:vcap /var/vcap

WORKDIR /var/vcap/jobs/stackit-cpi/bin

# Copy binary from builder
COPY --from=builder /go/bin/stackit-cpi ./stackit-cpi

# Make binary executable and set ownership
RUN chmod +x ./stackit-cpi && chown vcap:vcap ./stackit-cpi

# Switch to non-root user
USER vcap

# Add health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD ./stackit-cpi info || exit 1

# Set entrypoint
ENTRYPOINT ["./stackit-cpi"]
