# ── build stage ──────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/ovum-mqtt ./cmd/ovum-mqtt

# ── runtime stage ─────────────────────────────────────────────────────────────
FROM scratch

COPY --from=build /out/ovum-mqtt /ovum-mqtt

ENTRYPOINT ["/ovum-mqtt"]
