FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/multica-mcp-server ./cmd/multica-mcp-server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/multica-mcp-server /usr/local/bin/multica-mcp-server
ENTRYPOINT ["multica-mcp-server"]
