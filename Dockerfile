FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=0.0.0
RUN CGO_ENABLED=0 go build \
  -ldflags="-s -w -X github.com/strider2038/multica-mcp/internal/version.Version=${VERSION}" \
  -o /bin/multica-mcp .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/multica-mcp /usr/local/bin/multica-mcp
ENTRYPOINT ["multica-mcp"]
